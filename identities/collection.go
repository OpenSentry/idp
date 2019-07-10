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
  Password      string          `json:"password"`
}

type PostIdentitiesRequest struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

type PostIdentitiesResponse struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

type PutIdentitiesRequest struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

type PutIdentitiesResponse struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

func debugLog(app string, event string, msg string, requestId string) {
  if requestId == "" {
    fmt.Println(fmt.Sprintf("[app:%s][event:%s] %s", app, event, msg))
    return;
  }
  fmt.Println(fmt.Sprintf("[app:%s][request-id:%s][event:%s] %s", app, requestId, event, msg))
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

    identityList, err := idpbe.FetchIdentitiesForSub(env.Driver, id)
    if err == nil {
      //n := env.Database[id]
      n := identityList[0]
      if id == n.Id {
        c.JSON(http.StatusOK, gin.H{
          "id": n.Id,
          "name": n.Name,
          "email": n.Email,
          "password": n.Password,
        })
        return
      }
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

    hashedPassword, err := idpbe.CreatePassword(input.Password)
    if err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    newIdentity := idpbe.Identity{
      Id: input.Id,
      Name: input.Name,
      Email: input.Email,
      Password: hashedPassword,
    }
    identityList, err := idpbe.CreateIdentities(env.Driver, newIdentity)
    if err != nil {
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    n := identityList[0]

    c.JSON(http.StatusOK, gin.H{
      "id": n.Id,
      "name": n.Name,
      "email": n.Email,
      "password": n.Password,
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
