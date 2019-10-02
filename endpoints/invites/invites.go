package invites

import (
  "time"
  "net/http"
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

    invitedByIdentityId := c.MustGet("sub").(string)
    dbHumans, err := idp.FetchHumansById(env.Driver, []string{invitedByIdentityId})
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    var invitedBy idp.Human
    if len(dbHumans) > 0 {
      invitedBy = dbHumans[0]
    }

    var handleRequest = func(iRequests []*utils.Request) {

      for _, request := range iRequests {
        r := request.Request.(client.CreateInvitesRequest)

        var invited idp.Human
        if r.Invited != "" {
          dbHumans, err := idp.FetchHumansById(env.Driver, []string{r.Invited})
          if err != nil {
            request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
            continue
          }
          if len(dbHumans) > 0 {
            invited := dbHumans[0]
            if invited.Email != r.Email {
              request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND) // FIXME better error
              continue
            }
          } else {
            request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
            continue
          }

        }

        log.WithFields(logrus.Fields{"fixme": 1}).Debug("Put invite expiration into config")

        newInvite := idp.Invite{
          Identity: idp.Identity{
            Issuer: "", // FIXME
            ExpiresAt: time.Now().Unix() + (60 * 60 * 24), // 24 hours
          },
          HintUsername: r.HintUsername,
          Invited: invited,
        }
        invite, _, _, err := idp.CreateInvite(env.Driver, newInvite, invitedBy, idp.Email{ Email:r.Email })
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
            Email: invite.SentTo.Email,
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
        log.WithFields(logrus.Fields{ "email":newInvite.SentTo.Email, "hint_username":newInvite.HintUsername, "id":newInvite.Invited }).Debug(err.Error())
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

      for _, request := range iRequests {

        var dbInvites []idp.Invite
        var err error

        if request.Request == nil {

          // Fetch all, that the token is allowed to.

          dbInvites, err = idp.FetchInvitesAll(env.Driver)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          var ok []client.Invite
          for _, i := range dbInvites {
            ok = append(ok, client.Invite{
              Id: i.Id,
              IssuedAt: i.IssuedAt,
              ExpiresAt: i.ExpiresAt,
              Email: i.SentTo.Email,
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

          if r.Id != "" {
            dbInvites, err = idp.FetchInvitesById(env.Driver, []string{r.Id})
            if err != nil {
              log.Debug(err.Error())
              request.Response = utils.NewInternalErrorResponse(request.Index)
              continue
            }

            if len(dbInvites) > 0 {
              i := dbInvites[0]
              ok := []client.Invite{ {
                Id: i.Id,
                IssuedAt: i.IssuedAt,
                ExpiresAt: i.ExpiresAt,
                Email: i.SentTo.Email,
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