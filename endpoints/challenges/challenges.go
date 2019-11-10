package challenges

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

type ConfirmTemplateData struct {
  Challenge string
  Id string
  Code string
  Sender string
  Email string
}

func GetChallenges(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetChallenges",
    })

    var requests []client.ReadChallengesRequest
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

      // requestor := c.MustGet("sub").(string)
      // var requestedBy *idp.Identity
      // if requestor != "" {
      //  identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
      //  if err != nil {
      //    bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
      //    log.Debug(err.Error())
      //    return
      //  }
      //  if len(identities) > 0 {
      //    requestedBy = &identities[0]
      //  }
      // }

      for _, request := range iRequests {

        var dbChallenges []idp.Challenge
        var err error
        var ok client.ReadChallengesResponse

        if request.Input == nil {
          dbChallenges, err = idp.FetchChallenges(tx, nil)
        } else {
          r := request.Input.(client.ReadChallengesRequest)
          log = log.WithFields(logrus.Fields{"otp_challenge": r.OtpChallenge})
          dbChallenges, err = idp.FetchChallenges(tx, []idp.Challenge{ {Id: r.OtpChallenge} })
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

        if len(dbChallenges) > 0 {
          for _, d := range dbChallenges {
            ok = append(ok, client.Challenge{
              OtpChallenge: d.Id,
              Subject: d.Subject,
              Audience: d.Audience,
              IssuedAt: d.IssuedAt,
              ExpiresAt: d.ExpiresAt,
              TTL: d.ExpiresAt - d.IssuedAt,
              RedirectTo: d.RedirectTo,
              CodeType: d.CodeType,
              VerifiedAt: d.VerifiedAt,
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
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

func PostChallenges(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostChallenges",
    })

    var requests []client.CreateChallengesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

      // requestor := c.MustGet("sub").(string)
      // var requestedBy *idp.Identity
      // if requestor != "" {
      //  identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
      //  if err != nil {
      //    bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
      //    log.Debug(err.Error())
      //    return
      //  }
      //  if len(identities) > 0 {
      //    requestedBy = &identities[0]
      //  }
      // }

      for _, request := range iRequests {
        r := request.Input.(client.CreateChallengesRequest)

        newChallenge := idp.Challenge{
          JwtRegisteredClaims: idp.JwtRegisteredClaims{
            Subject: r.Subject,
            Issuer: config.GetString("idp.public.issuer"),
            Audience: config.GetString("idp.public.url") + config.GetString("idp.public.endpoints.challenges.verify"),
            ExpiresAt: time.Now().Unix() + r.TTL,
          },
          RedirectTo: r.RedirectTo,
          CodeType: r.CodeType,
        }

        var otpCode idp.ChallengeCode
        var challenge idp.Challenge
        if client.OTPType(newChallenge.CodeType) == client.TOTP {
          challenge, err = idp.CreateChallengeForTOTP(tx, newChallenge)
        } else {
          challenge, otpCode, err = idp.CreateChallengeForOTP(tx, newChallenge)
        }
        if err == nil && challenge.Id != "" {

          if otpCode.Code != "" && r.Email != "" {

            // Sent challenge to requested email

            var templateFile string
            var emailSubject string
            var sender idp.SMTPSender

            switch (r.Template) {
            case client.ConfirmEmail:
              sender = idp.SMTPSender{ Name: config.GetString("emailconfirm.sender.name"), Email: config.GetString("emailconfirm.sender.email") }
              templateFile = config.GetString("emailconfirm.template.email.file")
              emailSubject = config.GetString("emailconfirm.template.email.subject")
            case client.ConfirmDelete:
              sender = idp.SMTPSender{ Name: config.GetString("delete.sender.name"), Email: config.GetString("delete.sender.email") }
              templateFile = config.GetString("delete.template.email.file")
              emailSubject = config.GetString("delete.template.email.subject")
            default:
              sender = idp.SMTPSender{ Name: config.GetString("otp.sender.name"), Email: config.GetString("otp.sender.email") }
              templateFile = config.GetString("otp.template.email.file")
              emailSubject = config.GetString("otp.template.email.subject")
            }

            smtpConfig := idp.SMTPConfig{
              Host: config.GetString("mail.smtp.host"),
              Username: config.GetString("mail.smtp.user"),
              Password: config.GetString("mail.smtp.password"),
              Sender: sender,
              SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
            }

            var data = ConfirmTemplateData{
              Challenge: challenge.Id,
              Sender: sender.Name,
              Id: challenge.Subject,
              Email: r.Email,
              Code: otpCode.Code, // Note this is the clear text generated code and not the hashed one stored in DB.
            }
            _, err = idp.SendEmailUsingTemplate(smtpConfig, r.Email, r.Email, emailSubject, templateFile, data)
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

          }

          request.Output = bulky.NewOkResponse(request.Index, client.CreateChallengesResponse{
            OtpChallenge: challenge.Id,
            Subject: challenge.Subject,
            Audience: challenge.Audience,
            IssuedAt: challenge.IssuedAt,
            ExpiresAt: challenge.ExpiresAt,
            TTL: challenge.ExpiresAt - challenge.IssuedAt,
            RedirectTo: challenge.RedirectTo,
            CodeType: challenge.CodeType,
            Code: challenge.Code,
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
        log.Debug(err.Error())
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

