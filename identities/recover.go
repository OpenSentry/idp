package identities

import (
  "net/http"
  "fmt"

  "github.com/gin-gonic/gin"

  _ "golang-idp-be/config"
  _ "golang-idp-be/gateway/hydra"
)

type RecoverRequest struct {
  Id              string            `json:"id"`
}

type RecoverResponse struct {
  Id              string          `json:"id"`
  Email           string          `json:"email"`
  RecoverMethod   string          `json:"recover_method"`
}

func PostRecover(env *IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.PostRecover]", c.MustGet("RequestId")))
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
