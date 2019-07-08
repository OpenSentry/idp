package identities

import (
  "net/http"
  "fmt"

  "github.com/gin-gonic/gin"
)

type RevokeRequest struct {
  Id		string		`json:"id"`
}

type RevokeResponse struct {
  Id		string		`json:"id"`
  Revoked	bool		`json:"revoked"`
}

func PostRevoke(env *IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.PostRevoke]", c.MustGet("RequestId")))
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
