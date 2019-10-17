package invites

import (
  "net/url"
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

type InviteTemplateData struct {
  Id string
  Email string
  InvitationUrl string
  IdentityProvider string
}

func PostInvitesSend(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvitesSend",
    })

    var requests []client.CreateInvitesSendRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    sender := idp.SMTPSender{
      Name: config.GetString("provider.name"),
      Email: config.GetString("provider.email"),
    }

    smtpConfig := idp.SMTPConfig{
      Host: config.GetString("mail.smtp.host"),
      Username: config.GetString("mail.smtp.user"),
      Password: config.GetString("mail.smtp.password"),
      Sender: sender,
      SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
    }

    emailTemplateFile := config.GetString("invite.template.email.file")
    emailSubject := config.GetString("invite.template.email.subject")

    epInviteUrl, err := url.Parse( config.GetString("invite.url") )
    if err != nil {
      c.AbortWithStatus(http.StatusInternalServerError)
      log.Debug("Invalid invite.url in config. Hint: See config documentation.")
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

      for _, request := range iRequests {
        r := request.Input.(client.CreateInvitesSendRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        dbInvites, err := idp.FetchInvites(tx, requestedBy, []idp.Invite{ {Identity:idp.Identity{Id:r.Id}} })
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.WithFields(logrus.Fields{ "id":r.Id }).Debug(err.Error())
          return
        }

        if len(dbInvites) > 0 {
          invite := dbInvites[0]

          u := epInviteUrl // Copy already parsed url
          q := u.Query()
          q.Add("id", invite.Id)
          u.RawQuery = q.Encode()

          data := InviteTemplateData{
            Id: invite.Id,
            Email: invite.Email,
            InvitationUrl: u.String(),
            IdentityProvider: config.GetString("provider.name"),
          }

          _, err = idp.SendEmailUsingTemplate(smtpConfig, invite.Email, invite.Email, emailSubject, emailTemplateFile, data)
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
            log.WithFields(logrus.Fields{ "id": invite.Id, "file": emailTemplateFile }).Debug(err.Error())
            return
          }

          updatedInvite, err := idp.UpdateInviteSentAt(tx, requestedBy, invite)
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

          if updatedInvite != (idp.Invite{}) {
            request.Output = bulky.NewOkResponse(request.Index, client.CreateInvitesResponse{
              Id: updatedInvite.Id,
              IssuedAt: updatedInvite.IssuedAt,
              ExpiresAt: updatedInvite.ExpiresAt,
              Email: updatedInvite.Email,
              Username: updatedInvite.Username,
            })
            idp.EmitEventInviteSent(env.Nats, idp.Invite{Identity:idp.Identity{Id:updatedInvite.Id}})
            continue
          }
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
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
