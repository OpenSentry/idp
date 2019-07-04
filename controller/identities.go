package controller

import (
  _ "os"
  "fmt"
  "net/http"

  _ "golang.org/x/net/context"
  _ "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"

  oidc "github.com/coreos/go-oidc"
  "github.com/gin-gonic/gin"

  _ "golang-idp-be/config"
)

type IdpBeEnv struct {
  Provider *oidc.Provider
  HydraConfig *clientcredentials.Config
}

func GetIdentities(env *IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:GetIdentities]", c.MustGet("RequestId")))

    id, _ := c.GetQuery("id")

    if id == "user-1" {
      c.JSON(http.StatusOK, gin.H{
        "id": id,
        "name": "Test bruger",
        "email": "test@test.dk",
      })
      c.Abort()
      return
    }

    // Deny by default
    c.JSON(http.StatusNotFound, gin.H{
      "error": "Not found",
    })
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

/*
func GetIdentities(c *gin.Context) {
  fmt.Println(fmt.Sprintf("[request-id:%s][event:GetIdentities]", c.MustGet("RequestId")))

  id, _ := c.GetQuery("id")

  if id == "user-1" {
    c.JSON(http.StatusOK, gin.H{
      "id": id,
      "name": "Test bruger",
      "email": "test@test.dk",
    })
    c.Abort()
    return
  }

  // Deny by default
  c.JSON(http.StatusNotFound, gin.H{
    "error": "Not found",
  })
  c.Abort()
}*/

func PostIdentities(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
  })
}

func PutIdentities(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
  })
}

func PostIdentitiesRevoke(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
  })
}

func PostIdentitiesRecover(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
  })
}
