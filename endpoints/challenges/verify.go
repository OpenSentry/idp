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

      requestedByIdentityId := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Input.(client.UpdateChallengesVerifyRequest)

        var aChallenge []idp.Challenge
        aChallenge = append(aChallenge, idp.Challenge{Id: r.OtpChallenge})
        dbChallenges, err := idp.FetchChallenges(env.Driver, aChallenge)
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }
        if dbChallenges == nil {
          log.WithFields(logrus.Fields{ "otp_challenge": r.OtpChallenge, "id": requestedByIdentityId, }).Debug("Challenge not found")
          request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
          continue
        }
        challenge := dbChallenges[0]

        identities, err := idp.FetchHumansById(env.Driver, []string{challenge.Subject})
        if err != nil {
          log.WithFields(logrus.Fields{ "otp_challenge": challenge.Id, "id": challenge.Subject, }).Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }
        if identities == nil {
          log.WithFields(logrus.Fields{ "otp_challenge": challenge.Id, "id": challenge.Subject, }).Debug("Human not found")
          request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }
        identity := identities[0]

        var valid bool = false

        if challenge.CodeType == "TOTP" {

          if identity.TotpRequired == true {

            decryptedSecret, err := idp.Decrypt(identity.TotpSecret, config.GetString("totp.cryptkey"))
            if err != nil {
              log.WithFields(logrus.Fields{"otp_challenge": challenge.Id}).Debug(err.Error())
              request.Output = bulky.NewInternalErrorResponse(request.Index)
              continue
            }

            valid, _ = idp.ValidateOtp(r.Code, decryptedSecret)

          } else {
            log.WithFields(logrus.Fields{ "otp_challenge": challenge.Id, "id": challenge.Subject, }).Debug("TOTP not required for Human")
            request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_TOTP_NOT_REQUIRED)
            continue
          }

        } else {

          valid, _ = idp.ValidatePassword(challenge.Code, r.Code)

        }

        var ok client.UpdateChallengesVerifyResponse

        if valid == true {
          verifiedChallenge, err := idp.VerifyChallenge(env.Driver, challenge, requestedByIdentityId)
          if err != nil {
            log.WithFields(logrus.Fields{"otp_challenge": challenge.Id}).Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          ok = client.UpdateChallengesVerifyResponse{
            OtpChallenge: verifiedChallenge.Id,
            Verified: true,
            RedirectTo: verifiedChallenge.RedirectTo,
          }
        } else {

          // Deny by default
          ok = client.UpdateChallengesVerifyResponse{
            OtpChallenge: challenge.Id,
            Verified: false,
            RedirectTo: challenge.RedirectTo,
          }
        }

        request.Output = bulky.NewOkResponse(request.Index, ok)

      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
