package identities

import (
  "net/http"

  "github.com/gin-gonic/gin"

  //"golang-idp-be/config"
  "golang-idp-be/environment"
  //"golang-idp-be/gateway/idpbe"
  //"golang-idp-be/gateway/hydra"
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
    requestId := c.MustGet("RequestId").(string)
    environment.DebugLog(route.LogId, "PostRevoke", "", requestId)

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
