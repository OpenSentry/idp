package resourceservers

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  //"github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"

  bulky "github.com/charmixer/bulky/server"
)

func GetResourceServers(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetResourceServers",
    })

    var requests []client.ReadResourceServersRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {
      var resourceServers []idp.ResourceServer

      for _, request := range iRequests {
        if request.Input != nil {
          r := request.Input.(client.ReadResourceServersRequest)
          resourceServers = append(resourceServers, idp.ResourceServer{ Identity: idp.Identity{Id:r.Id} })
        }
      }

      dbResourceServers, err := idp.FetchResourceServers(env.Driver, resourceServers)
      if err != nil {
        log.Debug(err.Error())
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        return
      }

      log.Debug(dbResourceServers)

      for _, request := range iRequests {
        var r client.ReadResourceServersRequest
        if request.Input != nil {
          r = request.Input.(client.ReadResourceServersRequest)
        }

        var ok client.ReadResourceServersResponse
        for _, d := range dbResourceServers {
          if request.Input != nil && d.Id != r.Id {
            continue
          }

          // Translate from db model to rest model
          ok = append(ok, client.ResourceServer{
            Id: d.Id,
            Name: d.Name,
            Description: d.Description,
            Audience: d.Audience,
          })
        }

        request.Output = bulky.NewOkResponse(request.Index, ok)
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
