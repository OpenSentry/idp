package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "golang-idp-be/config"
  "golang-idp-be/environment"
  "golang-idp-be/gateway/idpapi"
)

type RecoverVerificationRequest struct {
  Challenge        string `json:"recover_challenge" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  Password         string  `json:"password" binding:"required"`
}

type RecoverVerificationResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verifed" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

func PostRecoverVerification(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRecoverVerification",
    })

    var input RecoverVerificationRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    denyResponse := RecoverVerificationResponse{
      Id: "",
      Verified: false,
      RedirectTo: "",
    }

    claims := idpapi.RecoverChallengeClaim{}
    token, err := idpapi.VerifyRecoverChallenge(input.Challenge, &claims, config.GetString("recover.sign.verify.path"))
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, denyResponse)
      c.Abort()
      return
    }

    if !token.Valid {
      log.WithFields(logrus.Fields{"recover_challenge": input.Challenge}).Debug("Invalid recover_challenge")
      c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid recover_challenge"})
      c.Abort();
      return
    }

    valid, err := idpapi.ValidatePassword(claims.VerificationCode, input.VerificationCode)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid verification code"})
      c.Abort();
      return
    }

    if valid == true {

      identities, err := idpapi.FetchIdentitiesForSub(env.Driver, claims.Subject)
      if err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid subject"})
        c.Abort();
        return
      }

      if identities == nil {
        log.WithFields(logrus.Fields{"id": claims.Subject}).Debug("Identity not found")
        c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
        c.Abort();
        return
      }

      identity := identities[0];

      // Update the password
      hashedPassword, err := idpapi.CreatePassword(input.Password)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }
      identity.Password = hashedPassword

      _, err = idpapi.UpdatePassword(env.Driver, identity)
      if err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Update password failed"})
        c.Abort();
        return
      }

      acceptResponse := RecoverVerificationResponse{
        Id: identity.Id,
        Verified: true,
        RedirectTo: "/me",
      }
      log.WithFields(logrus.Fields{
        "id": denyResponse.Id,
        "verified": denyResponse.Verified,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Recover accepted")
      c.JSON(http.StatusOK, acceptResponse)
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{
      "id": denyResponse.Id,
      "verified": denyResponse.Verified,
      "redirect_to": denyResponse.RedirectTo,
    }).Debug("Recover rejected")
    c.JSON(http.StatusOK, denyResponse)
  }
  return gin.HandlerFunc(fn)
}
