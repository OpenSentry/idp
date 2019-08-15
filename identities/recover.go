package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "golang-idp-be/environment"
)

type RecoverRequest struct {
  Id              string            `json:"id"`
}

type RecoverResponse struct {
  Id              string          `json:"id"`
  Email           string          `json:"email"`
  RecoverMethod   string          `json:"recover_method"`
}

func PostRecover(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{      
      "func": "PostRecover",
    })

    var input RecoverRequest
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
