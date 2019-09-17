package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

func PostRecoverVerification(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRecoverVerification",
    })

    var input IdentitiesRecoverVerificationRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    denyResponse := IdentitiesRecoverVerificationResponse{
      Id: input.Id,
      Verified: false,
      RedirectTo: "",
    }

    identity, exists, err := idp.FetchIdentityById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid id"})
      c.Abort();
      return
    }

    if exists == false {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      c.Abort();
      return
    }

    valid, err := idp.ValidatePassword(identity.OtpRecoverCode, input.VerificationCode)
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
      hashedPassword, err := idp.CreatePassword(input.Password)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      n := idp.Identity{
        Id: identity.Id,
        Password: hashedPassword,
      }
      updatedIdentity, err := idp.UpdatePassword(env.Driver, n)
      if err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Update password failed"})
        c.Abort();
        return
      }

      acceptResponse := IdentitiesRecoverVerificationResponse{
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
