package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "golang-idp-be/environment"
  "golang-idp-be/gateway/idpapi"
)

type IdentitiesRequest struct {
  Id            string          `json:"id" binding:"required"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

type IdentitiesResponse struct {
  Id            string          `json:"id" binding:"required"`
  Password      string          `json:"password" binding:"required"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

func GetCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetCollection",
    })

    id, _ := c.GetQuery("id")
    if id == "" {
      c.JSON(http.StatusNotFound, gin.H{
        "error": "Not found",
      })
      c.Abort()
      return;
    }

    identityList, err := idpapi.FetchIdentitiesForSub(env.Driver, id)
    if err == nil {
      n := identityList[0]
      if id == n.Id {
        c.JSON(http.StatusOK, IdentitiesResponse{
          Id: n.Id,
          Name: n.Name,
          Email: n.Email,
          Password: n.Password,
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

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostCollection",
    })

    var input IdentitiesRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    hashedPassword, err := idpapi.CreatePassword(input.Password)
    if err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    newIdentity := idpapi.Identity{
      Id: input.Id,
      Name: input.Name,
      Email: input.Email,
      Password: hashedPassword,
    }
    identityList, err := idpapi.CreateIdentities(env.Driver, newIdentity)
    if err != nil {
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    n := identityList[0]

    c.JSON(http.StatusOK, IdentitiesResponse{
      Id: n.Id,
      Name: n.Name,
      Email: n.Email,
      Password: n.Password,
    })
  }
  return gin.HandlerFunc(fn)
}

func PutCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    // Warning: Do not log user passwords!
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutCollection",
    })

    var input IdentitiesRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    updateIdentity := idpapi.Identity{
      Id: input.Id,
      Name: input.Name,
      Email: input.Email,
    }
    identityList, err := idpapi.UpdateIdentities(env.Driver, updateIdentity)
    if err != nil {
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    n := identityList[0]

    c.JSON(http.StatusOK, IdentitiesResponse{
      Id: n.Id,
      Name: n.Name,
      Email: n.Email,
      Password: n.Password,
    })
  }
  return gin.HandlerFunc(fn)
}

func DeleteCollection(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteCollection",
    })

    var input IdentitiesRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    deleteIdentity := idpapi.Identity{
      Id: input.Id,
    }
    deletedIdentity, err := idpapi.DeleteIdentity(env.Driver, deleteIdentity)
    if err != nil {
      log.WithFields(logrus.Fields{
        "id": input.Id,
      }).Debug(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    c.JSON(http.StatusOK, IdentitiesResponse{
      Id: deletedIdentity.Id,
    })
  }
  return gin.HandlerFunc(fn)
}
