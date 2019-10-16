package challenges

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
)

func PutVerify(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutVerify",
    })

    var requests []client.UpdateChallengesVerifyRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      //iRequest := idp.Identity{ Id: c.MustGet("sub").(string) }

      session, tx, err := idp.BeginWriteTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      for _, request := range iRequests {
        r := request.Input.(client.UpdateChallengesVerifyRequest)

        // Sanity check. Challenge must exists
        var aChallenge []idp.Challenge
        aChallenge = append(aChallenge, idp.Challenge{Id: r.OtpChallenge})
        dbChallenges, err := idp.FetchChallenges(tx, aChallenge)
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }

          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.WithFields(logrus.Fields{ "otp_challenge":r.OtpChallenge }).Debug(err.Error())
          return
        }

        cnt := len(dbChallenges)
        if cnt <= 0 {
          request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
          continue
        }
        if cnt > 1 {
          request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND) // FIXME: To many challenges, should never happen on primary key
          continue
        }

        var challenge idp.Challenge = dbChallenges[0]
        var valid bool = false

        if client.OTPType(challenge.CodeType) == client.TOTP {

          humans, err := idp.FetchHumansById(env.Driver, []string{challenge.Subject})
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
            log.WithFields(logrus.Fields{ "otp_challenge":challenge.Id, "id":challenge.Subject }).Debug(err.Error())
            return
          }

          cnt := len(humans)
          if cnt <= 0 {
            request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
            continue
          }
          if cnt > 1 {
            request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND) // FIXME: To many humans, should never happen on primary key
            continue
          }
          var human idp.Human = humans[0]

          if human.TotpRequired != true {
            request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_TOTP_NOT_REQUIRED)
            continue
          }

          decryptedSecret, err := idp.Decrypt(human.TotpSecret, config.GetString("totp.cryptkey"))
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
            log.WithFields(logrus.Fields{ "otp_challenge":challenge.Id, "id":human.Id }).Debug(err.Error())
            return
          }

          valid, _ = idp.ValidateOtp(r.Code, decryptedSecret)

        } else {

          valid, _ = idp.ValidatePassword(challenge.Code, r.Code)

        }

        if valid == true {

          verifiedChallenge, err := idp.VerifyChallenge(tx, challenge)
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
            log.WithFields(logrus.Fields{ "otp_challenge":challenge.Id }).Debug(err.Error())
            return
          }

          request.Output = bulky.NewOkResponse(request.Index, client.UpdateChallengesVerifyResponse{
            OtpChallenge: verifiedChallenge.Id,
            Verified: true,
            RedirectTo: verifiedChallenge.RedirectTo,
          })
          continue
        }

        // Deny by default
        request.Output = bulky.NewOkResponse(request.Index, client.UpdateChallengesVerifyResponse{
          OtpChallenge: r.OtpChallenge,
          Verified: false,
          RedirectTo: "",
        })
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
        return
      }
      tx.Rollback() // deny by default
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
