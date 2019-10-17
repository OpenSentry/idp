package humans

import (
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

type RecoverTemplateData struct {
  Name string
  VerificationCode string
  Sender string
}

func PostRecover(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRecover",
    })

    var requests []client.CreateHumansRecoverRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    sender := idp.SMTPSender{
      Name: config.GetString("recover.sender.name"),
      Email: config.GetString("recover.sender.email"),
    }

    smtpConfig := idp.SMTPConfig{
      Host: config.GetString("mail.smtp.host"),
      Username: config.GetString("mail.smtp.user"),
      Password: config.GetString("mail.smtp.password"),
      Sender: sender,
      SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
    }

    templateFile := config.GetString("recover.template.email.file")
    emailSubject := config.GetString("recover.template.email.subject")

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
        r := request.Input.(client.CreateHumansRecoverRequest)
        
        log = log.WithFields(logrus.Fields{"id": r.Id})

        dbHumans, err := idp.FetchHumans(tx, []idp.Human{ {Identity:idp.Identity{Id:r.Id}} })
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
          human := dbHumans[0]

          // FIXME: Use challenge system!

          challenge, err := idp.CreateRecoverChallenge(config.GetString("recover.link"), human, 60 * 5) // Fixme configfy 60*5
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          hashedCode, err := idp.CreatePassword(challenge.Code)
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

          n := idp.Human{
            Identity: idp.Identity{
              Id: human.Id,
            },
            OtpRecoverCode: hashedCode,
            OtpRecoverCodeExpire: challenge.Expire,
          }
          updatedHuman, err := idp.UpdateOtpRecoverCode(tx, n)
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

          if updatedHuman != (idp.Human{}) {
            data := RecoverTemplateData{
              Sender: sender.Name,
              Name: updatedHuman.Id,
              VerificationCode: challenge.Code,
            }
            _, err = idp.SendEmailUsingTemplate(smtpConfig, updatedHuman.Name, updatedHuman.Email, emailSubject, templateFile, data)
            if err != nil {
              e := tx.Rollback()
              if e != nil {
                log.Debug(e.Error())
              }
              bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
              request.Output = bulky.NewInternalErrorResponse(request.Index)
              log.WithFields(logrus.Fields{ "id": updatedHuman.Id, "file": templateFile }).Debug(err.Error())
              return
            }

            request.Output = bulky.NewOkResponse(request.Index, client.CreateHumansRecoverResponse{
              Id: updatedHuman.Id,
              RedirectTo: challenge.RedirectTo,
            })
            continue
          }

        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
        log.Debug("Recover human failed. Hint: Maybe input validation needs to be improved.")
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
