package invites

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

func PostInvites(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvites",
    })

    var input InviteCreateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    // Sanity check. InvitedBy
    invitedByIdentity, exists, err := idp.FetchIdentityById(env.Driver, input.InvitedBy)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if exists == false {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: invited_by"})
      return
    }

    // Sanity check. Invited Identity
    invitedIdentityId := ""
    if input.Invited != "" {
      identity, exists, err := idp.FetchIdentityById(env.Driver, input.Invited)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }
      if exists == false {
        c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: invited"})
        return
      }
      invitedIdentityId = identity.Id
    }


    log.WithFields(logrus.Fields{"fixme": 1}).Debug("Put invite expiration into config")
    identityInvite := idp.IdentityInvite{
      Email: input.Email,
      Username: input.Username,
      TTL: 60 * 60 * 24, // 24 hour invite
      InvitedBy: invitedByIdentity.Id,
      InvitedIdentityId: invitedIdentityId,
    }
    invite, err := idp.CreateIdentityInvite(env.Driver, identityInvite)
    if err != nil {
      log.WithFields(logrus.Fields{
        "id": identityInvite.Id,
        "email": identityInvite.Email,
      }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    response := InviteCreateResponse{ InviteResponse: marshalIdentityInviteToInviteResponse(invite) }

    log.WithFields(logrus.Fields{
      "id": response.Id,
    }).Debug("Invite send")
    c.JSON(http.StatusOK, response)
    return
  }
  return gin.HandlerFunc(fn)
}

func GetInvites(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetInvites",
    })

    var request InviteReadRequest
    err := c.BindJSON(&request)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    invite, exists, err := idp.FetchIdentityInviteById(env.Driver, request.Id)
    if err != nil {
      log.WithFields(logrus.Fields{"id": request.Id}).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if exists == true {
      c.JSON(http.StatusOK, InviteReadResponse{ marshalIdentityInviteToInviteResponse(invite) })
      return
    }

    // Deny by default
    c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Invite not found"})
    c.Abort()
  }
  return gin.HandlerFunc(fn)
}

func PutInvite(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutInvite",
    })

    var input IdentitiesInviteUpdateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    invite, exists, err := idp.FetchInviteById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if exists == true {

      // Created granted relations as specified in the invite
      // Create follow relations as specified in the invite
      accept, err := idp.AcceptInvite(env.Driver, invite)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return;
      }

      response := IdentitiesInviteUpdateResponse{
        IdentitiesInviteResponse: &IdentitiesInviteResponse{
          Id: accept.Id,
        },
      }
      log.WithFields(logrus.Fields{
        "id": accept.Id,
      }).Debug("Invite accepted")
      c.JSON(http.StatusOK, response)
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Invite not found")
    c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Invite not found"})
  }
  return gin.HandlerFunc(fn)
}

func marshalIdentityInviteToInviteResponse(invite idp.IdentityInvite) *InviteResponse {
  return &InviteResponse{
    Id: invite.Id,
    InvitedBy: invite.InvitedBy,
    TTL: invite.TTL,
    IssuedAt: invite.IssuedAt,
    ExpiresAt: invite.ExpiresAt,
    Email: invite.Email,
    Username: invite.Username,
    Invited: invite.InvitedIdentityId,
  }
}