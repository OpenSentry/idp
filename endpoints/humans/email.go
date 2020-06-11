package humans

import (
  "net/http"
  "context"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/opensentry/idp/app"
  "github.com/opensentry/idp/gateway/idp"
  "github.com/opensentry/idp/client"
  E "github.com/opensentry/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
)

func PutEmail(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

		ctx := context.TODO()

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutEmail",
    })

    var requests []client.UpdateHumansEmailRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      tx, err := env.Driver.BeginTx(ctx, nil)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }

      requestor := c.MustGet("sub").(string)
        var requestedBy *idp.Identity
        if requestor != "" {
        identities, err := idp.FetchIdentities(ctx, tx, []idp.Identity{ {Id:requestor} })
        if err != nil {
          bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
          log.Debug(err.Error())
          return
        }
        if len(identities) > 0 {
          requestedBy = &identities[0]
        }
      }

      for _, request := range iRequests {
        r := request.Input.(client.UpdateHumansEmailRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        // Sanity check. Do not allow updating on anything but the access token subject
        if requestedBy.Id != r.Id {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewErrorResponse(request.Index, http.StatusForbidden, E.HUMAN_TOKEN_INVALID)
          return
        }

        dbHumans, err := idp.FetchHumans(ctx, tx, []idp.Human{ {Identity:idp.Identity{Id:r.Id}} })
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

        if len(dbHumans) <= 0 {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          return
        }
        human := dbHumans[0]

        updatedHuman, err := idp.UpdateEmail(ctx, tx, idp.Human{ Identity: idp.Identity{ Id:  human.Id }, Email: r.Email })
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

        if updatedHuman != (idp.Human{}) {
          request.Output = bulky.NewOkResponse(request.Index, client.UpdateHumansEmailResponse{
            Id: updatedHuman.Id,
            Username: updatedHuman.Username,
            //Password: updatedHuman.Password,
            Name: updatedHuman.Name,
            Email: updatedHuman.Email,
            AllowLogin: updatedHuman.AllowLogin,
            TotpRequired: updatedHuman.TotpRequired,
            TotpSecret: updatedHuman.TotpSecret,
          })
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index)
        log.Debug("Update email failed. Hint: Maybe input validation needs to be improved.")
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
