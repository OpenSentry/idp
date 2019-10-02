package clients

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  //"github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"
  "github.com/charmixer/idp/utils"
)

func GetClients(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetClients",
    })

    var requests []client.ReadClientsRequest
    err := c.BindJSON(&requests)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*utils.Request) {
      var clients []idp.Client

      for _, request := range iRequests {
        if request.Request != nil {
          var r client.ReadClientsRequest
          r = request.Request.(client.ReadClientsRequest)
          clients = append(clients, idp.Client{ Identity: idp.Identity{Id:r.Id} })
        }
      }

      dbClients, err := idp.FetchClients(env.Driver, clients)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      var mapClients map[string]*idp.Client
      if ( iRequests[0] != nil ) {
        for _, client := range dbClients {
          mapClients[client.Id] = &client
        }
      }

      for _, request := range iRequests {

        if request.Request == nil {

          // The empty fetch
          var ok []client.Client
          for _, i := range dbClients {
            ok = append(ok, client.Client{
              Id: i.Id,
              ClientSecret: i.ClientSecret,
              Name: i.Name,
              Description: i.Description,
            })
          }
          var response client.ReadClientsResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue

        } else {

          r := request.Request.(client.ReadClientsRequest)

          var i = mapClients[r.Id]
          if i != nil {
            ok := []client.Client{ {
                Id: i.Id,
                ClientSecret: i.ClientSecret,
                Name: i.Name,
                Description: i.Description,
              },
            }
            var response client.ReadClientsResponse
            response.Index = request.Index
            response.Status = http.StatusOK
            response.Ok = ok
            request.Response = response
            continue
          }

        }

        // Deny by default
        request.Response = utils.NewClientErrorResponse(request.Index, E.CLIENT_NOT_FOUND)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequests, utils.HandleBulkRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}