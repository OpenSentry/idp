package invites

import (
  "time"
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
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

    issuer := config.GetString("idp.public.issuer")
    if issuer == "" {
      log.Debug("Missing idp.public.issuer")
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    ttl := config.GetInt("invite.ttl")
    if ttl <= 0 {
      log.Debug("Missing invite.ttl. Hint: Invites that never expire are not supported.")
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.CreateInvitesRequest)

        if env.BannedUsernames[r.Username] == true {
          request.Output = bulky.NewClientErrorResponse(request.Index, E.USERNAME_BANNED)
          continue
        }

        // Sanity check. Username must be unique
        if r.Username != "" {
          humansFoundByUsername, err := idp.FetchHumansByUsername(env.Driver, []string{r.Username})
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }
          if len(humansFoundByUsername) > 0 {
            request.Output = bulky.NewClientErrorResponse(request.Index, E.USERNAME_EXISTS)
            continue
          }
        }

        // Sanity check. Email must be unique
        dbHumans, err := idp.FetchHumansByEmail(env.Driver, []string{r.Email})
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }
        if len(dbHumans) > 0 {
          request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_ALREADY_EXISTS)
          continue
        }

        if (time.Now().Unix() >= r.ExpiresAt) {
          request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_EXPIRES_IN_THE_PAST)
          continue
        }

        var exp int64
        if r.ExpiresAt > 0 {
          exp = r.ExpiresAt
        } else {
          exp = time.Now().Unix() + int64(ttl)
        }

        newInvite := idp.Invite{
          Identity: idp.Identity{
            Issuer: issuer,
            ExpiresAt: exp,
          },
          Email: r.Email,
          Username: r.Username,
        }
        invite, err := idp.CreateInvite(env.Driver, newInvite)
        if err != nil {
          log.WithFields(logrus.Fields{ "email": r.Email }).Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if invite != (idp.Invite{}) {

          ok := client.CreateInvitesResponse{
            Id: invite.Id,
            IssuedAt: invite.IssuedAt,
            ExpiresAt: invite.ExpiresAt,
            Email: invite.Email,
            Username: invite.Username,
            SentAt: invite.SentAt,
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          log.WithFields(logrus.Fields{ "id":ok.Id }).Debug("Invite created")
          idp.EmitEventInviteCreated(env.Nats, invite)
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
          } else if r.Email != "" {
            dbInvites, err = idp.FetchInvitesByEmail(env.Driver, []string{r.Email})
          }
          // else if r.Username != "" {
          //   dbInvites, err = idp.FetchInvitesByUsername(env.Driver, []string{r.Username})
          // }

          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

        }

        if len(dbInvites) > 0 {

          for _, i := range dbInvites {
            ok = append(ok, client.Invite{
              Id: i.Id,
              IssuedAt: i.IssuedAt,
              ExpiresAt: i.ExpiresAt,
              Email: i.Email,
              SentAt: i.SentAt,
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