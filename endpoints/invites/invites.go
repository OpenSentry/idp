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

  bulky "github.com/charmixer/bulky/server"
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

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.CreateInvitesRequest)

        var invited idp.Human
        if r.Invited != "" {
          dbHumans, err := idp.FetchHumansById(env.Driver, []string{r.Invited})
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }
          if len(dbHumans) > 0 {
            invited := dbHumans[0]
            if invited.Email != r.Email {
              request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND) // TODO: Make better error
              continue
            }
          } else {
            request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
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
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if invite != (idp.Invite{}) {

          ok := client.CreateInvitesResponse{
            Id: invite.Id,
            IssuedAt: invite.IssuedAt,
            ExpiresAt: invite.ExpiresAt,
            Email: invite.SentTo.Email,
            Invited: invite.Invited.Id,
            HintUsername: invite.HintUsername,
            InvitedBy: invite.InvitedBy.Id,
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          log.WithFields(logrus.Fields{ "id": ok.Id, }).Debug("Invite created")
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_NOT_CREATED)
        continue
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
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
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {

        var dbInvites []idp.Invite
        var err error
        var ok client.ReadInvitesResponse

        if request.Input == nil {

          // Fetch all, that the token is allowed to.
          dbInvites, err = idp.FetchInvitesAll(env.Driver)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

        } else {

          r := request.Input.(client.ReadInvitesRequest)
          if r.Id != "" {
            dbInvites, err = idp.FetchInvitesById(env.Driver, []string{r.Id})
            if err != nil {
              log.Debug(err.Error())
              request.Output = bulky.NewInternalErrorResponse(request.Index)
              continue
            }
          }

        }

        if len(dbInvites) > 0 {

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

          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
        continue
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}