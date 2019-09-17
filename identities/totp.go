package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

type TotpResponse struct {
  Id string `json:"id" binding:"required"`
}

func PutTotp(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostTotp",
    })

    var input IdentitiesTotpRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    identity, exists, err := idp.FetchIdentityById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    if exists == true {

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

      c.JSON(http.StatusOK, IdentitiesReadResponse{ marshalIdentityToIdentityResponse(updatedIdentity) })
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Identity not found")
    c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
  }
  return gin.HandlerFunc(fn)
}
