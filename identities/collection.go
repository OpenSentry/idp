package identities

import (
  "net/http"

  "github.com/gin-gonic/gin"

  "golang-idp-be/environment"
  _ "golang-idp-be/config"
  _ "golang-idp-be/gateway/hydra"
  "golang-idp-be/gateway/idpbe"
)

type IdentitiesResponse struct {
  Id            string          `json:"id" binding:"required"`
  Password      string          `json:"password" binding:"required"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type IdentitiesRequest struct {
  Id            string          `json:"id" binding:"required"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

type GetIdentitiesRequest struct {
  *IdentitiesRequest
}

type GetIdentitiesResponse struct {
  *IdentitiesResponse
}

type PostIdentitiesRequest struct {
  *IdentitiesRequest
}

type PostIdentitiesResponse struct {
  *IdentitiesResponse
}

type PutIdentitiesRequest struct {
  *IdentitiesRequest
}

type PutIdentitiesResponse struct {
  *IdentitiesResponse
}

func GetCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    requestId := c.MustGet(environment.RequestIdKey).(string)
    environment.DebugLog(route.LogId, "GetCollection", "", requestId)

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

func PostCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    requestId := c.MustGet(environment.RequestIdKey).(string)
    environment.DebugLog(route.LogId, "PostCollection", "", requestId)

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

func PutCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    requestId := c.MustGet(environment.RequestIdKey).(string)
    environment.DebugLog(route.LogId, "PutCollection", "", requestId)

    var input PutIdentitiesRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    updateIdentity := idpbe.Identity{
      Id: input.Id,
      Name: input.Name,
      Email: input.Email,
    }
    identityList, err := idpbe.UpdateIdentities(env.Driver, updateIdentity)
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
