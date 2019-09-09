package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  //hydra "github.com/charmixer/hydra/client"

  //"github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  //"github.com/charmixer/idp/gateway/idp"
)

type IdentitiesInviteRequest struct {
  Id string `json:"id" binding:"required"`
  Permission []string
  RecommendFollow []string
  Email string
}

type IdentitiesInviteResponse struct {
  Id string `json:"id" binding:"required"`
}

func PostInvite(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvite",
    })

    var input IdentitiesInviteRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    //c.JSON(http.StatusOK, IdentitiesReadResponse{ marshalIdentityToIdentityResponse(identity) })

    // Deny by default
    log.WithFields(logrus.Fields{
      "id": "?",
    }).Debug("Invite not allowed")
    c.JSON(http.StatusNotFound, gin.H{"error": "Invite not allowed"})
  }
  return gin.HandlerFunc(fn)
}
