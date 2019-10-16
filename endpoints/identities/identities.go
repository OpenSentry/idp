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
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
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

    var handleRequests = func(iRequests []*bulky.Request) {
      //iRequest := idp.Identity{ Id: c.MustGet("sub").(string) }

      session, tx, err := idp.BeginReadTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      for _, request := range iRequests {

        var dbIdentities []idp.Identity
        var err error
        var ok client.ReadIdentitiesResponse

        if request.Input == nil {
          dbIdentities, err = idp.FetchIdentities(tx, nil)
        } else {
          r := request.Input.(client.ReadIdentitiesRequest)
          dbIdentities, err = idp.FetchIdentities(tx, []idp.Identity{ {Id: r.Id} })
        }
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if len(dbIdentities) > 0 {
          for _, i := range dbIdentities {
            ok = append(ok, client.Identity{
              Id: i.Id,
              Labels: strings.Split(i.Labels, ":"),
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}