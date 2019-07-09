package identities

import (
  "fmt"
  "net/http"

  _ "golang.org/x/net/context"
  _ "golang.org/x/oauth2"
  _ "golang.org/x/oauth2/clientcredentials"

  "github.com/gin-gonic/gin"

  _ "golang-idp-be/config"
  _ "golang-idp-be/gateway/hydra"
  "golang-idp-be/gateway/idpbe"
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


func GetCollection(env *idpbe.IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.GetCollection]", c.MustGet("RequestId")))

    id, _ := c.GetQuery("id")
    if id == "" {
      c.JSON(http.StatusNotFound, gin.H{
        "error": "Not found",
      })
      c.Abort()
      return;
    }

    n := env.Database[id]
    if id == n.Id {
      c.JSON(http.StatusOK, gin.H{
        "id": n.Id,
        "name": n.Name,
        "email": n.Email,
      })
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

func PostCollection(env *idpbe.IdpBeEnv) gin.HandlerFunc {
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

func PutCollection(env *idpbe.IdpBeEnv) gin.HandlerFunc {
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
