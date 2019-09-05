package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
)

type PasswordRequest struct {
  Id              string            `json:"id" binding:"required"`
  Password        string            `json:"password" binding:"required"`
}

type PasswordResponse struct {
  Id              string            `json:"id" binding:"required"`
}

func PutPassword(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostPassword",
    })

    var input PasswordRequest
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

      valid, _ := idp.ValidatePassword(identity.Password, input.Password)
      if valid == true {
        // Nothing to change was the new password is same as current password
        c.JSON(http.StatusOK, gin.H{"id": identity.Id})
        return
      }

      hashedPassword, err := idp.CreatePassword(input.Password)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      updatedIdentity, err := idp.UpdatePassword(env.Driver, idp.Identity{
        Id: identity.Id,
        Password: hashedPassword,
      })
      if err != nil {
        log.Debug(err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return;
      }

      c.JSON(http.StatusOK, gin.H{"id": updatedIdentity.Id})
      return
    }

    // Deny by default
    log.Info("Identity '"+input.Id+"' not found")
    c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
  }
  return gin.HandlerFunc(fn)
}
