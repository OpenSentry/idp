package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  //"github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  "github.com/charmixer/idp/utils"
)

func GetIdentities(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetIdentities",
    })

    var requests []client.ReadIdentitiesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*utils.Request) {
      var identities []idp.Identity

      for _, request := range iRequests {
        if request.Request != nil {
          var r client.ReadIdentitiesRequest
          r = request.Request.(client.ReadIdentitiesRequest)
          identities = append(identities, idp.Identity{ Id:r.Id })
        }
      }

      dbIdentities, err := idp.FetchIdentities(env.Driver, identities)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      // TODO: To decrease memory usage fix to use a pointer instead.
      var mapIdentities map[string]idp.Identity
      if ( iRequests[0] != nil ) {
        for _, identity := range dbIdentities {
          mapIdentities[identity.Id] = identity
        }
      }

      var ok []client.Identity

      for _, request := range iRequests {
        var r client.ReadIdentitiesRequest
        if request.Request != nil {  // Allow empty request
          r = request.Request.(client.ReadIdentitiesRequest)

          var i = mapIdentities[r.Id]
          ok = append(ok, client.Identity{
            Id: i.Id,
          })

        } else {
          for _, d := range dbIdentities {
            ok = append(ok, client.Identity{
              Id: d.Id,
            })
          }

        }

        var response client.ReadIdentitiesResponse
        response.Index = request.Index
        response.Status = http.StatusOK
        response.Ok = ok
        request.Response = response
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequests, utils.HandleBulkRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}