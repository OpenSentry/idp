package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
)

type TotpRequest struct {
  Id          string `json:"id" binding:"required"`
  TotpRequired bool   `json:"totp_required,omitempty" binding:"required"`
  TotpSecret   string `json:"totp_secret,omitempty" binding:"required"`
}

type TotpResponse struct {
  Id string `json:"id" binding:"required"`
}

func PutTotp(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostTotp",
    })

    var input TotpRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    identities, err := idp.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    if identities != nil {

      identity := identities[0]; // FIXME do not return a list of identities!

      encryptedSecret, err := idp.Encrypt(input.TotpSecret, config.GetString("totp.cryptkey"))
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      updatedIdentity, err := idp.UpdateTotp(env.Driver, idp.Identity{
        Id: identity.Id,
        TotpRequired: input.TotpRequired,
        TotpSecret: encryptedSecret,
      })
      if err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return;
      }

      c.JSON(http.StatusOK, TotpResponse{ Id: updatedIdentity.Id })
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Identity not found")
    c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
  }
  return gin.HandlerFunc(fn)
}
