package invites

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"
  "github.com/charmixer/idp/utils"
)

func PutInvitesAccept(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutInvitesAccept",
    })

    var requests []client.UpdateInvitesAcceptRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequest = func(iRequests []*utils.Request) {

      //requestedByIdentity := c.MustGet("sub").(string)

      var invites []idp.Invite
      for _, request := range iRequests {
        if request.Request != nil {
          var r client.UpdateInvitesAcceptRequest
          r = request.Request.(client.UpdateInvitesAcceptRequest)
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
        r := request.Request.(client.UpdateInvitesAcceptRequest)

        var i = mapInvites[r.Id]
        if i != nil {
          accept, err := idp.AcceptInvite(env.Driver, *i)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          ok := []client.Invite{ {
              Id: accept.Id,
              IssuedAt: accept.IssuedAt,
              //ExpiresAt: accept.ExpiredAt,
              Email: accept.Email,
              Invited: accept.Invited.Id,
              HintUsername: accept.HintUsername,
              InvitedBy: accept.InvitedBy.Id,
            },
          }

          var response client.UpdateInvitesAcceptResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          log.WithFields(logrus.Fields{ "id": accept.Id, }).Debug("Invite accepted")
          continue
        }

        // Deny by default
        request.Response = utils.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
