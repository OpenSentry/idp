package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "golang-idp-be/environment"
  "golang-idp-be/gateway/idpapi"
)

type RecoverVerificationRequest struct {
  Id               string `json:"id" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  Password         string `json:"password" binding:"required"`
  RedirectTo       string `json:"redirect_to" binding:"required"`
}

type RecoverVerificationResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verified" binding:"required"`
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
      Id: input.Id,
      Verified: false,
      RedirectTo: "",
    }

    identities, err := idpapi.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid id"})
      c.Abort();
      return
    }

    if identities == nil {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      c.Abort();
      return
    }

    identity := identities[0];

    valid, err := idpapi.ValidatePassword(identity.OtpRecoverCode, input.VerificationCode)
    if err != nil {
      log.Debug(err.Error())
      log.WithFields(logrus.Fields{
        "id": denyResponse.Id,
        "verified": denyResponse.Verified,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Recover rejected")
      c.JSON(http.StatusOK, denyResponse)
      c.Abort();
      return
    }

    if valid == true {

      // Update the password
      hashedPassword, err := idpapi.CreatePassword(input.Password)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      n := idpapi.Identity{
        Id: identity.Id,
        Password: hashedPassword,
      }
      updatedIdentity, err := idpapi.UpdatePassword(env.Driver, n)
      if err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Update password failed"})
        c.Abort();
        return
      }

      acceptResponse := RecoverVerificationResponse{
        Id: updatedIdentity.Id,
        Verified: true,
        RedirectTo: input.RedirectTo,
      }
      log.WithFields(logrus.Fields{
        "id": acceptResponse.Id,
        "verified": acceptResponse.Verified,
        "redirect_to": acceptResponse.RedirectTo,
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
