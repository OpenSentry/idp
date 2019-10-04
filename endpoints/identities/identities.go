package identities

import (
  "net/http"
  "strings"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  //"github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  "github.com/charmixer/idp/utils"
  E "github.com/charmixer/idp/client/errors"
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

      for _, request := range iRequests {

        var dbIdentities []idp.Identity
        var err error

        if request.Request == nil {

          dbIdentities, err = idp.FetchIdentitiesAll(env.Driver)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          // The empty fetch
          var ok []client.Identity
          for _, i := range dbIdentities {
            ok = append(ok, client.Identity{
              Id: i.Id,
              Labels: strings.Split(i.Labels, ":"),
            })
          }
          var response client.ReadIdentitiesResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue

        } else {

          r := request.Request.(client.ReadIdentitiesRequest)

          dbIdentities, err = idp.FetchIdentitiesById(env.Driver, []string{r.Id})
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          if len(dbIdentities) > 0 {
            i := dbIdentities[0]
            ok := []client.Identity{{
              Id: i.Id,
              Labels: strings.Split(i.Labels, ":"),
            }}
            var response client.ReadIdentitiesResponse
            response.Index = request.Index
            response.Status = http.StatusOK
            response.Ok = ok
            request.Response = response
            continue
          }

        }

        // Deny by default
        request.Response = utils.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequests, utils.HandleBulkRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}