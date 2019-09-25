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
    var invitedByIdentity idp.Human
    invitedByIdentities, err := idp.FetchHumansById(env.Driver, []string{input.InvitedByIdentity})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    if invitedByIdentities == nil {
      c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: invited_by"})
      return
    }
    invitedByIdentity = invitedByIdentities[0]


    // Sanity check. Invited Identity
    var invitedIdentity idp.Human
    if input.InvitedIdentity != "" {
      identities, err := idp.FetchHumansById(env.Driver, []string{input.InvitedIdentity})
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }
      if identities == nil {
        c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Indentity not found. Hint: invited"})
        return
      }
      invitedIdentity = identities[0]
    }

    log.WithFields(logrus.Fields{"fixme": 1}).Debug("Put invite expiration into config")
    invite := idp.Invite{
      HintUsername: input.HintUsername,
      Human: idp.Human{
        Identity: idp.Identity{
          Issuer: "",
          ExpiresAt: 60 * 60 * 24, // 24 hours
        },
      },

      Invited: &invitedIdentity,
    }
    invite, _, _, err = idp.CreateInvite(env.Driver, invite, invitedByIdentity, idp.Email{ Email:input.Email })
    if err != nil {
      log.WithFields(logrus.Fields{
        "invited_by": invitedByIdentity.Id,
        "email": input.Email,
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

    var requests []InviteReadRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }


    invites, err := idp.FetchInvites(env.Driver, nil)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if len(invites) > 0 {
      c.JSON(http.StatusOK, InviteReadResponse{ marshalIdentityInviteToInviteResponse(invites[0]) })
      return
    }

    // Deny by default
    c.AbortWithStatusJSON(http.StatusOK, []InviteReadResponse{})
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

    invites, err := idp.FetchInvites(env.Driver, nil)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    if invites != nil {

      accept, err := idp.AcceptInvite(env.Driver, invites[0])
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

func marshalIdentityInviteToInviteResponse(invite idp.Invite) *InviteResponse {
  return &InviteResponse{
    Id: invite.Id,
    InvitedBy: invite.InvitedBy.Id,
    TTL: invite.ExpiresAt - invite.IssuedAt,
    IssuedAt: invite.IssuedAt,
    ExpiresAt: invite.ExpiresAt,
    Email: invite.SentTo.Email,
    Username: invite.Username,
    Invited: invite.Invited.Id,
  }
}