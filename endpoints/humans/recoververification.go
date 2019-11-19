package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/app"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
)

func PutRecoverVerification(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutRecoverVerification",
    })

    var requests []client.UpdateHumansRecoverVerifyRequest
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
        r := request.Input.(client.UpdateHumansRecoverVerifyRequest)

        log = log.WithFields(logrus.Fields{"recover_challenge": r.RecoverChallenge})

        dbChallenges, err := idp.FetchChallenges(tx, []idp.Challenge{ {Id: r.RecoverChallenge} })
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

        if len(dbChallenges) <= 0 {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
          return
        }

        challenge := dbChallenges[0]

        if challenge.VerifiedAt > 0 {

          // FIXME: We need to make sure the challenge was actually for a deletion else any challenge can be used.
          // -- solution could be to add a challenge_type to the challenge system {Login, EmailConfirmation, DeleteConfirmation, ...}

          // Challenge verified, delete human
          log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all access tokens for identity - put them on revoked list or rely on expire")
          log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all consents in hydra for identity - this is probably aap?")
          log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all sessions in hydra for identity - this is probably aap?")

          // Update the password
          hashedPassword, err := idp.CreatePassword(r.NewPassword)
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

          updatedHuman, err := idp.UpdatePassword(tx, idp.Human{ Identity: idp.Identity{ Id: challenge.Subject }, Password: hashedPassword })
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
            request.Output = bulky.NewOkResponse(request.Index, client.UpdateHumansRecoverVerifyResponse{
              Id: challenge.Subject,
              Verified: true,
              RedirectTo: challenge.RedirectTo,
            })
            continue
          }

        }

        // Deny by default
        request.Output = bulky.NewOkResponse(request.Index, client.UpdateHumansRecoverVerifyResponse{
          Id: challenge.Subject,
          Verified: false,
          RedirectTo: "",
        })
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

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
