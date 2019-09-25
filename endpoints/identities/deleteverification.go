package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

func PostDeleteVerification(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostDeleteVerification",
    })

    var input IdentitiesDeleteVerificationRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    denyResponse := IdentitiesDeleteVerificationResponse{
      Id: input.Id,
      Verified: false,
      RedirectTo: "",
    }

    identities, err := idp.FetchIdentitiesById(env.Driver, []string{input.Id})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if identities == nil {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
      return
    }
    identity := identities[0]

    valid, err := idp.ValidatePassword(identity.OtpDeleteCode, input.VerificationCode)
    if err != nil {
      log.Debug(err.Error())
      log.WithFields(logrus.Fields{
        "id": denyResponse.Id,
        "verified": denyResponse.Verified,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Delete verification rejected")
      c.JSON(http.StatusOK, denyResponse)
      return
    }

    if valid == true {

      log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all access tokens for identity - put them on revoked list or rely on expire")
      log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all consents in hydra for identity - this is probably aap?")

      n := idp.Human{
        Identity: idp.Identity{
          Id: identity.Id,
        },
      }
      updatedIdentity, err := idp.DeleteHuman(env.Driver, n)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      acceptResponse := IdentitiesDeleteVerificationResponse{
        Id: updatedIdentity.Id,
        Verified: true,
        RedirectTo: input.RedirectTo,
      }
      log.WithFields(logrus.Fields{
        "id": acceptResponse.Id,
        "verified": acceptResponse.Verified,
        "redirect_to": acceptResponse.RedirectTo,
      }).Debug("Identity deleted")
      c.JSON(http.StatusOK, acceptResponse)
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{
      "id": denyResponse.Id,
      "verified": denyResponse.Verified,
      "redirect_to": denyResponse.RedirectTo,
    }).Debug("Delete verification rejected")
    c.JSON(http.StatusOK, denyResponse)
  }
  return gin.HandlerFunc(fn)
}
