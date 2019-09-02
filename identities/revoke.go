package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  
  "github.com/charmixer/idp/environment"
)

type RevokeRequest struct {
  Id		string		`json:"id"`
}

type RevokeResponse struct {
  Id		string		`json:"id"`
  Revoked	bool		`json:"revoked"`
}

func PostRevoke(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRevoke",
    })

    var input RevokeRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    c.JSON(http.StatusOK, gin.H{
      "message": "pong",
    })
  }
  return gin.HandlerFunc(fn)
}
