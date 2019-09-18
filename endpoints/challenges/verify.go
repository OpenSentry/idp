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

func PostVerify(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostVerify",
    })

    var input VerifyRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    denyResponse := VerifyResponse{
      OtpChallenge: input.OtpChallenge,
      Verified: false,
      RedirectTo: "",
    }

    challenge, exists, err := idp.FetchChallenge(env.Driver, input.OtpChallenge)
    if err != nil {
      log.WithFields(logrus.Fields{
        "otp_challenge": input.OtpChallenge,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if exists == false {
      log.WithFields(logrus.Fields{
        "otp_challenge": input.OtpChallenge,
      }).Debug("Challenge not found")
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
      return
    }

    identity, exists, err := idp.FetchIdentityById(env.Driver, challenge.Subject)
    if err != nil {
      log.WithFields(logrus.Fields{
        "otp_challenge": challenge.OtpChallenge,
        "id": challenge.Subject,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if exists == false {
      log.WithFields(logrus.Fields{
        "otp_challenge": challenge.OtpChallenge,
        "id": challenge.Subject,
      }).Debug("Identity not found")
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      return
    }

    var valid bool = false

    if challenge.CodeType == "TOTP" {

      if identity.TotpRequired == true {

        decryptedSecret, err := idp.Decrypt(identity.TotpSecret, config.GetString("totp.cryptkey"))
        if err != nil {
          log.Debug(err.Error())
          c.AbortWithStatus(http.StatusInternalServerError)
          return
        }

        valid, _ = idp.ValidateOtp(input.Code, decryptedSecret)

      } else {
        log.WithFields(logrus.Fields{
          "otp_challenge": challenge.OtpChallenge,
          "id": challenge.Subject,
        }).Debug("TOTP not enabled for Identity")
        c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "TOTP not enabled for Identity. Hint: Use a code of digits instead."})
        return
      }

    } else {

      valid, _ = idp.ValidatePassword(challenge.Code, input.Code)

    }

    if valid == true {
      verifiedChallenge, exists, err := idp.VerifyChallenge(env.Driver, challenge)
      if err != nil {
        log.WithFields(logrus.Fields{"otp_challenge": challenge.OtpChallenge}).Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      if exists == false {
        log.WithFields(logrus.Fields{
          "otp_challenge": challenge.OtpChallenge,
          "id": challenge.Subject,
        }).Debug("Challenge not found")
        c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
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
