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

    type FollowCreateRequest struct {
      Id string `json:"id" binding:"required"`
      Follow string `json:"follow" binding:"required"`
    }

    // Sanity check. Identity that are to follow another identity must exist
    fromIdentity, exists, err := idp.FetchIdentityById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if exists == false {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: id"})
      return
    }

    // Sanity check. Identity that follows exists
    toIdentity, exists, err := idp.FetchIdentityById(env.Driver, input.Follow)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if exists == false {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: follow"})
      return
    }

    follow, err := idp.CreateFollow(env.Driver, fromIdentity, toIdentity)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    response := FollowCreateResponse{ FollowResponse: marshalFollowToFollowResponse(follow) }
    c.JSON(http.StatusOK, response)
  }
  return gin.HandlerFunc(fn)
}

func marshalFollowToFollowResponse(follow idp.Follow) *FollowResponse {
  return &FollowResponse{
    Id: follow.Id,
    Follow: follow.FollowIdentity,
  }
}