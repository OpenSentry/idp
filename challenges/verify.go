package challenges

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
)

type VerifyRequest struct {
  OtpChallenge string `json:"otp_challenge" binding:"required"`
  Code       string `json:"code" binding:"required"`
}

type VerifyResponse struct {
  OtpChallenge string `json:"otp_challenge" binding:"required"`
  Verified     bool   `json:"verified" binding:"required"`
  RedirectTo   string `json:"redirect_to" binding:"required"`
}

func PostVerify(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostVerify",
    })

    var input VerifyRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    denyResponse := VerifyResponse{
      OtpChallenge: input.OtpChallenge,
      Verified: false,
      RedirectTo: "",
    }

    challenge, err := idp.FetchChallenge(env.Driver, input.OtpChallenge)
    if err != nil {
      log.WithFields(logrus.Fields{
        "otp_challenge": input.OtpChallenge,
      }).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch OTP challenge"})
      c.Abort()
      return
    }

    if challenge == nil {
      log.WithFields(logrus.Fields{
        "otp_challenge": input.OtpChallenge,
      }).Debug("Challenge not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
      c.Abort()
      return
    }

    identities, err := idp.FetchIdentitiesForSub(env.Driver, challenge.Subject)
    if err != nil {
      log.WithFields(logrus.Fields{
        "otp_challenge": challenge.OtpChallenge,
        "id": challenge.Subject,
      }).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Identity"})
      c.Abort()
      return;
    }

    if identities == nil {
      log.WithFields(logrus.Fields{
        "otp_challenge": challenge.OtpChallenge,
        "id": challenge.Subject,
      }).Debug("Identity not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      c.Abort()
      return;
    }

    if len(identities) > 1 {
      log.WithFields(logrus.Fields{
        "otp_challenge": challenge.OtpChallenge,
        "id": challenge.Subject,
      }).Debug("Found to many identities")
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Found to many Identity"})
      c.Abort()
      return;
    }

    identity := identities[0];

    var valid bool = false

    if challenge.CodeType == "TOTP" {

      if identity.TotpRequired == true {

        decryptedSecret, err := idp.Decrypt(identity.TotpSecret, config.GetString("totp.cryptkey"))
        if err != nil {
          c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
          c.Abort()
          return
        }

        valid, _ = idp.ValidateOtp(input.Code, decryptedSecret)

      } else {
        log.WithFields(logrus.Fields{
          "otp_challenge": challenge.OtpChallenge,
          "id": challenge.Subject,
        }).Debug("TOTP not enabled for Identity")
        c.JSON(http.StatusNotFound, gin.H{"error": "TOTP not enabled for Identity. Hint: Use a code of digits instead."})
        c.Abort()
        return;
      }

    } else {

      valid, _ = idp.ValidatePassword(challenge.Code, input.Code)

    }

    if valid == true {
      verifiedChallenge, err := idp.VerifyChallenge(env.Driver, *challenge)
      if err != nil {
        log.Debug(err.Error())
        log.WithFields(logrus.Fields{"otp_challenge": challenge.OtpChallenge}).Debug("Set challenge verified failed")
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      c.JSON(http.StatusOK, VerifyResponse{
        OtpChallenge: verifiedChallenge.OtpChallenge,
        Verified: true,
        RedirectTo: verifiedChallenge.RedirectTo,
      })
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{
      "otp_challenge": denyResponse.OtpChallenge,
      "verified": denyResponse.Verified,
      "redirect_to": denyResponse.RedirectTo,
    }).Debug("Verify denied")
    c.JSON(http.StatusOK, denyResponse)
  }
  return gin.HandlerFunc(fn)
}
