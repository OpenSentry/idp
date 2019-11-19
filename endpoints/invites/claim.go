package invites

import (
  "net/http"
  "net/url"
  "time"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/app"
  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
)

type ConfirmTemplateData struct {
  Challenge string
  Id string
  Code string
  Sender string
  Email string
}

func PostInvitesClaim(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvitesClaim",
    })

    var requests []client.CreateInvitesClaimRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    epVerifyController := config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.emailconfirm")
    redirectToConfirm, err := url.Parse(epVerifyController)
    if err != nil {
      c.AbortWithStatus(http.StatusInternalServerError)
      log.Debug(err.Error())
      return
    }

    var templateFile string
    var emailSubject string
    var sender idp.SMTPSender

    sender = idp.SMTPSender{ Name: config.GetString("provider.name"), Email: config.GetString("provider.email") }
    templateFile = config.GetString("templates.emailconfirm.email.templatefile")
    emailSubject = config.GetString("templates.emailconfirm.email.subject")

    smtpConfig := idp.SMTPConfig{
      Host: config.GetString("mail.smtp.host"),
      Username: config.GetString("mail.smtp.user"),
      Password: config.GetString("mail.smtp.password"),
      Sender: sender,
      SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
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

      // requestor := c.MustGet("sub").(string)
      // var requestedBy *idp.Identity
      // if requestor != "" {
      //   identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
      //   if err != nil {
      //     bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
      //     log.Debug(err.Error())
      //     return
      //   }
      //   if len(identities) > 0 {
      //     requestedBy = &identities[0]
      //   }
      // }

      for _, request := range iRequests {
        r := request.Input.(client.CreateInvitesClaimRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        // Create email claim challenge based of the invite
        redirectToUrlWhenVerified, err := url.Parse( r.RedirectTo ) // If this fails the input validation is bad and needs fixing
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

        inv := idp.Invite{ Identity: idp.Identity{ Id: r.Id } }
        dbInvites, err := idp.FetchInvites(tx, nil, []idp.Invite{ inv })
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

        if len(dbInvites) <= 0 {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
          return
        }

        invite := dbInvites[0]

        newChallenge := idp.Challenge{
          JwtRegisteredClaims: idp.JwtRegisteredClaims{
            Subject: invite.Id,
            Issuer: config.GetString("idp.public.issuer"),
            Audience: config.GetString("idp.public.url") + config.GetString("idp.public.endpoints.challenges.verify"),
            ExpiresAt: time.Now().Unix() + r.TTL,
          },
          RedirectTo: redirectToUrlWhenVerified.String(),
          CodeType: int64(client.OTP),
        }
        challenge, otpCode, err := idp.CreateChallengeUsingOtp(tx, idp.ChallengeEmailConfirm, newChallenge)
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

        if challenge != (idp.Challenge{}) {

          if otpCode.Code != "" && invite.Email != "" {

            var data = ConfirmTemplateData{
              Challenge: challenge.Id,
              Sender: sender.Name,
              Id: challenge.Subject,
              Email: invite.Email,
              Code: otpCode.Code, // Note this is the clear text generated code and not the hashed one stored in DB.
            }
            _, err = idp.SendEmailUsingTemplate(smtpConfig, invite.Email, invite.Email, emailSubject, templateFile, data)
            if err != nil {
              e := tx.Rollback()
              if e != nil {
                log.Debug(e.Error())
              }
              bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
              request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
              log.WithFields(logrus.Fields{"error": err.Error()}).Debug("Failed to send email")
              return
            }

          }

          q := redirectToConfirm.Query()
          q.Add("email_challenge", challenge.Id)
          redirectToConfirm.RawQuery = q.Encode()

          request.Output = bulky.NewOkResponse(request.Index, client.CreateInvitesClaimResponse{
            RedirectTo: redirectToConfirm.String(),
          })
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
        log.Debug("Create challenge failed. Hint: Maybe input validation needs to be improved.")
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
