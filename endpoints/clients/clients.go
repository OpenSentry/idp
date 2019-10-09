package clients

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

func GetClients(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetClients",
    })

    var requests []client.ReadChallengesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {
      var clients []idp.Client

      for _, request := range iRequests {
        if request.Input != nil {
          var r client.ReadClientsRequest
          r = request.Input.(client.ReadClientsRequest)
          clients = append(clients, idp.Client{ Identity: idp.Identity{Id:r.Id} })
        }
      }

      dbClients, err := idp.FetchClients(env.Driver, clients)
      if err != nil {
        log.Debug(err.Error())
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        return
      }

      for _, request := range iRequests {
        var r client.ReadClientsRequest
        if request.Input != nil {
          r = request.Input.(client.ReadClientsRequest)
        }

        var ok client.ReadClientsResponse
        for _, d := range dbClients {
          if request.Input != nil && d.Id != r.Id {
            continue
          }

          // Translate from db model to rest model
          ok = append(ok, client.Client{
            Id: d.Id,
            ClientSecret: d.ClientSecret,
            Name: d.Name,
            Description: d.Description,
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