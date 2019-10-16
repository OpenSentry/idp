package resourceservers

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

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

      session, tx, err := idp.BeginReadTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      // requestor := c.MustGet("sub").(string)
      // var requestedBy *idp.Identity
      // if requestor != "" {
      //  identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
      //  if err != nil {
      //    bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
      //    log.Debug(err.Error())
      //    return
      //  }
      //  if len(identities) > 0 {
      //    requestedBy = &identities[0]
      //  }
      // }

      for _, request := range iRequests {

        var dbResourceServers []idp.ResourceServer
        var err error
        var ok client.ReadResourceServersResponse

        if request.Input == nil {
          dbResourceServers, err = idp.FetchResourceServers(tx, nil)
        } else {
          r := request.Input.(client.ReadResourceServersRequest)
          dbResourceServers, err = idp.FetchResourceServers(tx, []idp.ResourceServer{ {Identity:idp.Identity{Id: r.Id}} })
        }
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }

        if len(dbResourceServers) > 0 {
          for _, d := range dbResourceServers {
            ok = append(ok, client.ResourceServer{
              Id: d.Id,
              Name: d.Name,
              Description: d.Description,
              Audience: d.Audience,
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)
        continue
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
