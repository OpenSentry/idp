package invites

import (
  "net/http"
  "time"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"
  "github.com/charmixer/idp/utils"
)

func PostInvites(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvites",
    })


    var requests []client.CreateInvitesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequest = func(iRequests []*utils.Request) {

      var humanIds []string

      invitedByIdentityId := c.MustGet("sub").(string)
      humanIds = append(humanIds, invitedByIdentityId)

      for _, request := range iRequests {
        if request.Request != nil {
          var r client.CreateInvitesRequest
          r = request.Request.(client.CreateInvitesRequest)
          humanIds = append(humanIds, r.Invited)
        }
      }

      dbHumans, err := idp.FetchHumansById(env.Driver, humanIds)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      var mapHumans map[string]*idp.Human
      if ( iRequests[0] != nil ) {
        for _, human := range dbHumans {
          mapHumans[human.Id] = &human
        }
      }

      for _, request := range iRequests {
        r := request.Request.(client.CreateInvitesRequest)

        invitedBy := mapHumans[invitedByIdentityId]
        if invitedBy == nil {
          request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }

        var invited idp.Human
        if r.Invited != "" {
          invited := mapHumans[r.Invited]
          if invited == nil {
            request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
            continue
          }

          if invited.Email != r.Email {
            request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND) // FIXME: Return different
            continue
          }
        }

        log.WithFields(logrus.Fields{"fixme": 1}).Debug("Put invite expiration into config")
        newInvite := idp.Invite{
          HintUsername: r.HintUsername,
          Human: idp.Human{
            Identity: idp.Identity{
              Issuer: "",
              ExpiresAt: 60 * 60 * 24, // 24 hours
            },
          },

          Invited: &invited,
        }
        invite, _, _, err := idp.CreateInvite(env.Driver, newInvite, *invitedBy, idp.Email{ Email:r.Email })
        if err != nil {
          log.WithFields(logrus.Fields{ "invited_by": invitedBy.Id, "email": r.Email }).Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if invite != (idp.Invite{}) {
          ok := client.Invite{
              Id: invite.Id,
              IssuedAt: invite.IssuedAt,
              ExpiresAt: invite.ExpiresAt,
              Email: invite.Email,
              Invited: invite.Invited.Id,
              HintUsername: invite.HintUsername,
              InvitedBy: invite.InvitedBy.Id,
          }
          var response client.CreateInvitesResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          log.WithFields(logrus.Fields{ "id": ok.Id, }).Debug("Invite created")
          continue
        }

        // Deny by default
        // @SecurityRisk: Please _NEVER_ log the hashed password
        log.WithFields(logrus.Fields{ "email":newInvite.Email, "hint_username":newInvite.HintUsername, "id":newInvite.Invited }).Debug(err.Error())
        request.Response = utils.NewClientErrorResponse(request.Index, E.INVITE_NOT_CREATED)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func GetInvites(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetInvites",
    })

    var requests []client.ReadInvitesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*utils.Request) {
      var invites []idp.Invite

      for _, request := range iRequests {
        if request.Request != nil {
          var r client.ReadInvitesRequest
          r = request.Request.(client.ReadInvitesRequest)
          invites = append(invites, idp.Invite{ Human:idp.Human{ Identity:idp.Identity{Id: r.Id}} })
        }
      }

      dbInvites, err := idp.FetchInvites(env.Driver, invites)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      var mapInvites map[string]*idp.Invite
      if ( iRequests[0] != nil ) {
        for _, invite := range dbInvites {
          mapInvites[invite.Id] = &invite
        }
      }

      for _, request := range iRequests {

        if request.Request == nil {

          // The empty fetch
          var ok []client.Invite
          for _, i := range dbInvites {
            ok = append(ok, client.Invite{
              Id: i.Id,
              IssuedAt: i.IssuedAt,
              //ExpiresAt: i.ExpiredAt,
              Email: i.Email,
              Invited: i.Invited.Id,
              HintUsername: i.HintUsername,
              InvitedBy: i.InvitedBy.Id,
            })
          }
          var response client.ReadInvitesResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue

        } else {

          r := request.Request.(client.ReadInvitesRequest)

          var i = mapInvites[r.Id]
          if i != nil {
            ok := []client.Invite{ {
                Id: i.Id,
                IssuedAt: i.IssuedAt,
                //ExpiresAt: i.ExpiredAt,
                Email: i.Email,
                Invited: i.Invited.Id,
                HintUsername: i.HintUsername,
                InvitedBy: i.InvitedBy.Id,
              },
            }
            var response client.ReadInvitesResponse
            response.Index = request.Index
            response.Status = http.StatusOK
            response.Ok = ok
            request.Response = response
            continue
          }

        }

        // Deny by default
        request.Response = utils.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequests, utils.HandleBulkRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}