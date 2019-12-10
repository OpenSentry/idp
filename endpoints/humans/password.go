package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/opensentry/idp/app"
  "github.com/opensentry/idp/gateway/idp"
  "github.com/opensentry/idp/client"
  E "github.com/opensentry/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
)

func PutPassword(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutPassword",
    })

    var requests []client.UpdateHumansPasswordRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      session, tx, err := idp.BeginWriteTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      requestor := c.MustGet("sub").(string)
        var requestedBy *idp.Identity
        if requestor != "" {
        identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
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
        r := request.Input.(client.UpdateHumansPasswordRequest)

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

        dbHumans, err := idp.FetchHumans(tx, []idp.Human{ {Identity:idp.Identity{Id:r.Id}} })
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

        valid, _ := idp.ValidatePassword(human.Password, r.Password)
        if valid == true {
          // Nothing to change was the new password is same as current password
          request.Output = bulky.NewOkResponse(request.Index, client.UpdateHumansPasswordResponse{
            Id: human.Id,
            Username: human.Username,
            Password: human.Password,
            Name: human.Name,
            Email: human.Email,
            AllowLogin: human.AllowLogin,
            TotpRequired: human.TotpRequired,
            TotpSecret: human.TotpSecret,
          })
          continue
        }

        hashedPassword, err := idp.CreatePassword(r.Password)
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

        updatedHuman, err := idp.UpdatePassword(tx, idp.Human{
          Identity: idp.Identity{
            Id: human.Id,
          },
          Password: hashedPassword,
        })
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
          request.Output = bulky.NewOkResponse(request.Index, client.UpdateHumansPasswordResponse{
            Id: updatedHuman.Id,
            Username: updatedHuman.Username,
            Password: updatedHuman.Password,
            Name: updatedHuman.Name,
            Email: updatedHuman.Email,
            AllowLogin: updatedHuman.AllowLogin,
            TotpRequired: updatedHuman.TotpRequired,
            TotpSecret: updatedHuman.TotpSecret,
          })
          idp.EmitEventHumanPasswordChanged(env.Nats, updatedHuman)
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index)
        log.Debug("Update password failed. Hint: Maybe input validation needs to be improved.")
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
