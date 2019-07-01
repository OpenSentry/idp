package main

import (
  "fmt"
  "strings"
  "net/http"

  "golang.org/x/oauth2"

  "github.com/gin-gonic/gin"
  "github.com/atarantini/ginrequestid"

  "golang-idp-be/config"
  "golang-idp-be/controller"
)

func init() {
  config.InitConfigurations()
}

func main() {

  r := gin.Default()
  r.Use(ginrequestid.RequestId())
  //r.Use(logRequest())
  r.Use(requireBearerAccessToken())
  r.GET( "/identities", controller.GetIdentities)
  r.POST("/identities", controller.PostIdentities)
  r.PUT( "/identities", controller.PutIdentities)
  r.POST( "/identities/authenticate", controller.PostIdentitiesAuthenticate)
  r.POST( "/identities/logout", controller.PostIdentitiesLogout)
  r.POST( "/identities/revoke", controller.PostIdentitiesRevoke)
  r.POST( "/identities/recover", controller.PostIdentitiesRecover)

  r.Run() // listen and serve on 0.0.0.0:8080
}

func logRequest() gin.HandlerFunc {
  return func(c *gin.Context) {
    fmt.Println("Logging all requests. Do not do this in production it will leak tokens")
    fmt.Println(c.Request)
  }
}

// Look for a bearer token and unmarshal it into the gin context for the request for later use.
func requireBearerAccessToken() gin.HandlerFunc {
  return func(c *gin.Context) {
    auth := c.Request.Header.Get("Authorization")
    split := strings.SplitN(auth, " ", 2)
    if len(split) == 2 && strings.EqualFold(split[0], "bearer") {
      token := &oauth2.Token{
        AccessToken: split[1],
        TokenType: split[0],
      }

      if token.Valid() {
        c.Set("bearer_token", token)
        c.Next()
        return
      }

      // Token invalid
      c.JSON(http.StatusForbidden, gin.H{"error": "Authorization bearer token is invalid"})
      c.Abort()
      return;
    }

    fmt.Println("wtf")

    // Deny by default.
    c.JSON(http.StatusForbidden, gin.H{"error": "Authorization bearer token is missing"})
    c.Abort()
  }
}
