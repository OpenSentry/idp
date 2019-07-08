package identities

import (
  "fmt"
  "net/http"

  _ "golang.org/x/net/context"
  _ "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"

  oidc "github.com/coreos/go-oidc"
  "github.com/gin-gonic/gin"

  _ "golang-idp-be/config"
)

type GetIdentitiesRequest struct {
  Id            string          `json:"id"`
}

type GetIdentitiesResponse struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type PostIdentitiesRequest struct {
  Id            string          `json:"id"`
  Password      string          `json:"password"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type PostIdentitiesResponse struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type PutIdentitiesRequest struct {
  Id            string          `json:"id"`
  Password      string          `json:"password"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type PutIdentitiesResponse struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type IdpBeEnv struct {
  Provider *oidc.Provider
  HydraConfig *clientcredentials.Config
}

func GetCollection(env *IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.GetCollection]", c.MustGet("RequestId")))

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

func PostCollection(env *IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.PostCollection]", c.MustGet("RequestId")))

    var input PostIdentitiesRequest
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

func PutCollection(env *IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.PutCollection]", c.MustGet("RequestId")))

    var input PutIdentitiesRequest
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
