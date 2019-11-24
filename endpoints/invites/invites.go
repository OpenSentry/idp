package invites

import (
  "time"
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/app"
  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

  aap "github.com/charmixer/aap/client"

  bulky "github.com/charmixer/bulky/server"
)

func PostInvites(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
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

      session, tx, err := idp.BeginWriteTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      requestor := c.MustGet("sub").(string)
      var requestedBy *idp.Identity
      if requestor != "" {
       identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
       if err != nil {
         bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
         log.Debug(err.Error())
         return
       }
       if len(identities) > 0 {
         requestedBy = &identities[0]
       }
      }

      var ids []string

      for _, request := range iRequests {
        r := request.Input.(client.CreateInvitesRequest)

        if env.BannedUsernames[r.Username] == true {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.USERNAME_BANNED)
          log.Debug(err.Error())
          return
        }

        // Sanity check. Username must be unique
        if r.Username != "" {
          humansFoundByUsername, err := idp.FetchHumansByUsername(tx, []idp.Human{ {Username:r.Username} })
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
            log.Debug(err.Error())
            return
          }
          if len(humansFoundByUsername) > 0 {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewClientErrorResponse(request.Index, E.USERNAME_EXISTS)
            log.Debug(err.Error())
            return
          }
        }

        // Sanity check. Email must be unique
        dbHumans, err := idp.FetchHumansByEmail(tx, []idp.Human{ {Email: r.Email} })
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }
        if len(dbHumans) > 0 {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_ALREADY_EXISTS)
          return
        }

        var exp int64
        if r.ExpiresAt > 0 {
          exp = r.ExpiresAt
        } else {
          exp = time.Now().Unix() + int64(ttl)
        }

        if (time.Now().Unix() >= exp) {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_EXPIRES_IN_THE_PAST)
          return
        }

        newInvite := idp.Invite{
          Identity: idp.Identity{
            Issuer: issuer,
            ExpiresAt: exp,
          },
          Email: r.Email,
          Username: r.Username,
        }
        log.Debug(newInvite)
        invite, err := idp.CreateInvite(tx, requestedBy, newInvite)
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }

        if invite != (idp.Invite{}) {
          ids = append(ids, invite.Id)

          request.Output = bulky.NewOkResponse(request.Index, client.CreateInvitesResponse{
            Id: invite.Id,
            IssuedAt: invite.IssuedAt,
            ExpiresAt: invite.ExpiresAt,
            Email: invite.Email,
            Username: invite.Username,
            SentAt: invite.SentAt,
          })
          idp.EmitEventInviteCreated(env.Nats, invite)
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
        log.Debug("Not able to create Invite. Hint. Maybe the input validation needs to be improved.")
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()

        var createEntitiesRequests []aap.CreateEntitiesRequest
        for _,id := range ids {
          createEntitiesRequests = append(createEntitiesRequests, aap.CreateEntitiesRequest{
            Reference: id,
            Creator: id,
            Scopes: []string{
              "aap:read:grants",
              "aap:create:grants",
              "aap:delete:grants",
              "aap:read:publishes",
              "aap:create:publishes",
              "aap:delete:publishes",
              "aap:read:subscriptions",
              "aap:create:subscriptions",
              "aap:delete:subscriptions",
              "aap:read:consents",
              "aap:create:consents",
              "aap:delete:consents",
              "aap:read:shadows",
              "aap:create:shadows",
              "aap:delete:shadows",

              "mg:aap:read:grants",
              "mg:aap:create:grants",
              "mg:aap:delete:grants",
              "mg:aap:read:publishes",
              "mg:aap:create:publishes",
              "mg:aap:delete:publishes",
              "mg:aap:read:subscriptions",
              "mg:aap:create:subscriptions",
              "mg:aap:delete:subscriptions",
              "mg:aap:read:consents",
              "mg:aap:create:consents",
              "mg:aap:delete:consents",
              "mg:aap:read:shadows",
              "mg:aap:create:shadows",
              "mg:aap:delete:shadows",

              "0:mg:aap:read:grants",
              "0:mg:aap:create:grants",
              "0:mg:aap:delete:grants",
              "0:mg:aap:read:publishes",
              "0:mg:aap:create:publishes",
              "0:mg:aap:delete:publishes",
              "0:mg:aap:read:subscriptions",
              "0:mg:aap:create:subscriptions",
              "0:mg:aap:delete:subscriptions",
              "0:mg:aap:read:consents",
              "0:mg:aap:create:consents",
              "0:mg:aap:delete:consents",
              "0:mg:aap:read:shadows",
              "0:mg:aap:create:shadows",
              "0:mg:aap:delete:shadows",
            },
          })
        }

        // Initialize in AAP model
        aapClient := aap.NewAapClient(env.AapConfig)
        url := config.GetString("aap.public.url") + config.GetString("aap.public.endpoints.entities.collection")
        status, response, err := aap.CreateEntities(aapClient, url, createEntitiesRequests)

        if err != nil {
          log.WithFields(logrus.Fields{ "error": err.Error(), "ids": ids }).Debug("Failed to initialize entity in AAP model")
        }

        log.WithFields(logrus.Fields{ "status": status, "response": response }).Debug("Initialize request for invites in AAP model")

        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func GetInvites(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
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

      session, tx, err := idp.BeginReadTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      requestor := c.MustGet("sub").(string)
      var requestedBy *idp.Identity
      if requestor != "" {
       identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
       if err != nil {
         bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
         log.Debug(err.Error())
         return
       }
       if len(identities) > 0 {
         requestedBy = &identities[0]
       }
      }

      for _, request := range iRequests {

        var dbInvites []idp.Invite
        var err error
        var ok client.ReadInvitesResponse

        if request.Input == nil {
          dbInvites, err = idp.FetchInvites(tx, requestedBy, nil)
        } else {
          r := request.Input.(client.ReadInvitesRequest)
          if r.Id != "" {
            dbInvites, err = idp.FetchInvites(tx, requestedBy, []idp.Invite{ {Identity:idp.Identity{Id:r.Id}} })
          } else if r.Email != "" {
            dbInvites, err = idp.FetchInvitesByEmail(tx, requestedBy, []idp.Invite{ {Email:r.Email} })
          }
          //else if r.Username != "" {
          //  dbInvites, err = idp.FetchInvitesByUsername(env.Driver, []string{r.Username})
          //}

        }
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }

        if len(dbInvites) > 0 {
          for _, i := range dbInvites {
            ok = append(ok, client.Invite{
              Id: i.Id,
              IssuedAt: i.IssuedAt,
              ExpiresAt: i.ExpiresAt,
              Email: i.Email,
              SentAt: i.SentAt,
              Username: i.Username,
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
        continue
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
