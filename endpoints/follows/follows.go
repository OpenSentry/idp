package follows

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

func PostFollows(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvites",
    })

    var input FollowCreateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    // Sanity check. Identity that are to follow another identity must exist
    fromIdentities, err := idp.FetchIdentitiesById(env.Driver, []string{input.Id})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if fromIdentities == nil {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: id"})
      return
    }
    fromIdentity := fromIdentities[0]

    // Sanity check. Identity that follows exists
    toIdentities, err := idp.FetchIdentitiesById(env.Driver, []string{input.Follow})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if toIdentities == nil {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: follow"})
      return
    }
    toIdentity := toIdentities[0]

    edge, err := idp.CreateFollow(env.Driver, fromIdentity, toIdentity)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    response := FollowCreateResponse{ FollowResponse: marshalFollowToFollowResponse(edge) }
    c.JSON(http.StatusOK, response)
  }
  return gin.HandlerFunc(fn)
}

func marshalFollowToFollowResponse(edge idp.Follow) *FollowResponse {
  return &FollowResponse{
    Id: edge.From.Id,
    Follow: edge.To.Id,
  }
}