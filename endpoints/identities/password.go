package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

func PutPassword(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostPassword",
    })

    var input IdentitiesPasswordRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    humans, err := idp.FetchHumansById(env.Driver, []string{input.Id})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if humans != nil {
      human := humans[0]

      valid, _ := idp.ValidatePassword(human.Password, input.Password)
      if valid == true {
        // Nothing to change was the new password is same as current password
        c.JSON(http.StatusOK, IdentitiesPasswordResponse{ marshalIdentityToIdentityResponse(human) })
        return
      }

      hashedPassword, err := idp.CreatePassword(input.Password)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      updatedIdentity, err := idp.UpdatePassword(env.Driver, idp.Human{
        Identity: idp.Identity{
          Id: human.Id,
        },
        Password: hashedPassword,
      })
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      c.JSON(http.StatusOK, IdentitiesReadResponse{ marshalIdentityToIdentityResponse(updatedIdentity) })
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Identity not found")
    c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
  }
  return gin.HandlerFunc(fn)
}
