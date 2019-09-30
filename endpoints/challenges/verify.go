package challenges

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  "github.com/charmixer/idp/utils"
  E "github.com/charmixer/idp/client/errors"
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

    var handleRequest = func(iRequests []*utils.Request) {

      requestedByIdentityId := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Request.(client.UpdateChallengesVerifyRequest)

        var aChallenge []idp.Challenge
        aChallenge = append(aChallenge, idp.Challenge{Id: r.OtpChallenge})
        dbChallenges, err := idp.FetchChallenges(env.Driver, aChallenge)
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }
        if dbChallenges == nil {
          log.WithFields(logrus.Fields{ "otp_challenge": r.OtpChallenge, "id": requestedByIdentityId, }).Debug("Challenge not found")
          request.Response = utils.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
          continue
        }
        challenge := dbChallenges[0]

        identities, err := idp.FetchHumansById(env.Driver, []string{challenge.Subject})
        if err != nil {
          log.WithFields(logrus.Fields{ "otp_challenge": challenge.Id, "id": challenge.Subject, }).Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }
        if identities == nil {
          log.WithFields(logrus.Fields{ "otp_challenge": challenge.Id, "id": challenge.Subject, }).Debug("Human not found")
          request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }
        identity := identities[0]

        var valid bool = false

        if challenge.CodeType == "TOTP" {

          if identity.TotpRequired == true {

            decryptedSecret, err := idp.Decrypt(identity.TotpSecret, config.GetString("totp.cryptkey"))
            if err != nil {
              log.WithFields(logrus.Fields{"otp_challenge": challenge.Id}).Debug(err.Error())
              request.Response = utils.NewInternalErrorResponse(request.Index)
              continue
            }

            valid, _ = idp.ValidateOtp(r.Code, decryptedSecret)

          } else {
            log.WithFields(logrus.Fields{ "otp_challenge": challenge.Id, "id": challenge.Subject, }).Debug("TOTP not required for Human")
            request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_TOTP_NOT_REQUIRED)
            continue
          }

        } else {

          valid, _ = idp.ValidatePassword(challenge.Code, r.Code)

        }

        if valid == true {
          verifiedChallenge, err := idp.VerifyChallenge(env.Driver, challenge, requestedByIdentityId)
          if err != nil {
            log.WithFields(logrus.Fields{"otp_challenge": challenge.Id}).Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          ok := client.ChallengeVerification{
            OtpChallenge: verifiedChallenge.Id,
            Verified: true,
            RedirectTo: verifiedChallenge.RedirectTo,
          }

          response := client.UpdateChallengesVerifyResponse{Ok: ok}
          response.Index = request.Index
          response.Status = http.StatusOK
          request.Response = response
        } else {
          // Deny by default
          ok := client.ChallengeVerification{
            OtpChallenge: challenge.Id,
            Verified: false,
            RedirectTo: challenge.RedirectTo,
          }

          response := client.UpdateChallengesVerifyResponse{Ok: ok}
          response.Index = request.Index
          response.Status = http.StatusOK
          request.Response = response
        }

      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{})

    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
