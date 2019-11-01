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

  aap "github.com/charmixer/aap/client"

  bulky "github.com/charmixer/bulky/server"
)

type DeleteTemplateData struct {
  Name string
  VerificationCode string
  Sender string
}

func GetHumans(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetHumans",
    })

    var requests []client.ReadHumansRequest
    err := c.BindJSON(&requests)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    // owners := c.Get("owners")

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

        var dbHumans []idp.Human
        var err error
        var ok client.ReadHumansResponse

        if request.Input == nil {
          dbHumans, err = idp.FetchHumans(tx, nil)
        } else {

          r := request.Input.(client.ReadHumansRequest)
          if r.Id != "" {
            log = log.WithFields(logrus.Fields{"id": r.Id})
            dbHumans, err = idp.FetchHumans(tx, []idp.Human{ {Identity:idp.Identity{Id:r.Id}} })
          } else if r.Email != "" {
            dbHumans, err = idp.FetchHumansByEmail(tx, []idp.Human{ {Email:r.Email} })
          } else if r.Username != "" {
            dbHumans, err = idp.FetchHumansByUsername(tx, []idp.Human{ {Username:r.Username} })
          }

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

        if len(dbHumans) > 0 {
          for _, i := range dbHumans {
            ok = append(ok, client.Human{
              Id:                   i.Id,
              Username:             i.Username,
              //Password:             i.Password,
              Name:                 i.Name,
              Email:                i.Email,
              AllowLogin:           i.AllowLogin,
              TotpRequired:         i.TotpRequired,
              TotpSecret:           i.TotpSecret,
              OtpRecoverCode:       i.OtpRecoverCode,
              OtpRecoverCodeExpire: i.OtpRecoverCodeExpire,
              OtpDeleteCode:        i.OtpDeleteCode,
              OtpDeleteCodeExpire:  i.OtpDeleteCodeExpire,
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue;
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
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

func PostHumans(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostHumans",
    })

    var requests []client.CreateHumansRequest
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

      var ids []string

      for _, request := range iRequests {
        r := request.Input.(client.CreateHumansRequest)

        if env.BannedUsernames[r.Username] == true {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.USERNAME_BANNED)
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
            request.Output = bulky.NewInternalErrorResponse(request.Index)
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
            return
          }
        }

        hashedPassword, err := idp.CreatePassword(r.Password) // @SecurityRisk: Please _NEVER_ log the cleartext password
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          log.Debug(err.Error())
          return
        }

        newHuman := idp.Human{
          Identity: idp.Identity{ Id: r.Id },
          Username: r.Username,
          Name: r.Name,
          Password: hashedPassword,
          AllowLogin: true,
          EmailConfirmedAt: r.EmailConfirmedAt,
        }
        human, err := idp.CreateHumanFromInvite(tx, newHuman)
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          // @SecurityRisk: Please _NEVER_ log the hashed password
          log.WithFields(logrus.Fields{
            "username": newHuman.Username,
            "name": newHuman.Name,
            "email": newHuman.Email,
            /* "password": newHuman.Password, */
            "allow_login": newHuman.AllowLogin,
            "email_confirmed_at": newHuman.EmailConfirmedAt,
          }).Debug(err.Error())
          return
        }

        if human != (idp.Human{}) {
          ids = append(ids, human.Id)

          ok := client.CreateHumansResponse{
              Id: human.Id,
              Username: human.Username,
              Password: human.Password,
              Name: human.Name,
              Email: human.Email,
              EmailConfirmedAt: human.EmailConfirmedAt,
              AllowLogin: human.AllowLogin,
              TotpRequired: human.TotpRequired,
              TotpSecret: human.TotpSecret,
              OtpRecoverCode: human.OtpRecoverCode,
              OtpRecoverCodeExpire: human.OtpRecoverCodeExpire,
              OtpDeleteCode: human.OtpDeleteCode,
              OtpDeleteCodeExpire: human.OtpDeleteCodeExpire,
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          log.WithFields(logrus.Fields{"id":ok.Id}).Debug("Human created")
          idp.EmitEventHumanCreated(env.Nats, human)
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index)
        // @SecurityRisk: Please _NEVER_ log the hashed password
        log.WithFields(logrus.Fields{
          "username": newHuman.Username,
          "name": newHuman.Name,
          "email": newHuman.Email,
          /* "password": newHuman.Password, */
          "allow_login": newHuman.AllowLogin,
          "email_confirmed_at": newHuman.EmailConfirmedAt,
        }).Debug("Not able to create Human from Invite. Hint: Maybe input validation needs to be improved.")
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()

        var createGrantsRequests []aap.CreateGrantsRequest
        for _,id := range ids {

          // TODO: Make this a role on IDP and just grant that.
          publisherId := config.GetString("id")

          // openid ?
          // offline ?
          // idp:update:humans:password ?
          // idp:update:humans:totp ?

          grantScopes := []string{
            "idp:create:humans:recover",
            "idp:delete:humans",
            "idp:update:humans",
            "idp:read:humans",
            "idp:create:humans:logout",
            "idp:read:humans:logout",
            "idp:update:humans:logout",
            "idp:read:resourceservers", // ?
            "idp:create:resourceservers", // ?
            "idp:delete:resourceservers", // ?
            "idp:create:clients",
            "idp:read:clients",
            "idp:delete:clients",
          }

          for _,s := range grantScopes {
            createGrantsRequests = append(createGrantsRequests, aap.CreateGrantsRequest{
              Identity: id,
              Scope: s,
              Publisher: publisherId,
              OnBehalfOf: id, // Only allow access to self
            })
          }

        }

        // Initialize in AAP model
        aapClient := aap.NewAapClient(env.AapConfig)
        url := config.GetString("aap.public.url") + config.GetString("aap.public.endpoints.grants")
        status, response, err := aap.CreateGrants(aapClient, url, createGrantsRequests)

        if err != nil {
          log.WithFields(logrus.Fields{ "error": err.Error(), "ids": ids }).Debug("Failed to initialize grants in AAP model")
        }

        log.WithFields(logrus.Fields{ "status": status, "response": response }).Debug("Initialize request for humans in AAP model")

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

func PutHumans(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutHumans",
    })

    var requests []client.UpdateHumansRequest
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
        r := request.Input.(client.UpdateHumansRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        updateHuman := idp.Human{
          Identity: idp.Identity{
            Id: r.Id,
          },
          Name: r.Name,
        }
        human, err := idp.UpdateHuman(tx, updateHuman)
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          log.Debug(err.Error())
          return
        }

        if human != (idp.Human{}) {
          ok := client.UpdateHumansResponse{
            Id: human.Id,
            Username: human.Username,
            //Password: human.Password,
            Name: human.Name,
            Email: human.Email,
            AllowLogin: human.AllowLogin,
            TotpRequired: human.TotpRequired,
            TotpSecret: human.TotpSecret,
            OtpRecoverCode: human.OtpRecoverCode,
            OtpRecoverCodeExpire: human.OtpRecoverCodeExpire,
            OtpDeleteCode: human.OtpDeleteCode,
            OtpDeleteCodeExpire: human.OtpDeleteCodeExpire,
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index)
        log.WithFields(logrus.Fields{
          "id": updateHuman.Id,
          "name": updateHuman.Name,
        }).Debug("Update human failed. Hint: Maybe input validation needs to be improved.")
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

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func DeleteHumans(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteHumans",
    })

    var requests []client.DeleteHumansRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    sender := idp.SMTPSender{
      Name: config.GetString("delete.sender.name"),
      Email: config.GetString("delete.sender.email"),
    }

    smtpConfig := idp.SMTPConfig{
      Host: config.GetString("mail.smtp.host"),
      Username: config.GetString("mail.smtp.user"),
      Password: config.GetString("mail.smtp.password"),
      Sender: sender,
      SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
    }

    templateFile := config.GetString("delete.template.email.file")
    emailSubject := config.GetString("delete.template.email.subject")

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
        r := request.Input.(client.DeleteHumansRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        dbHumans, err := idp.FetchHumans(tx, []idp.Human{ {Identity:idp.Identity{Id:r.Id}} })
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          log.Debug(err.Error())
          return
        }

        if len(dbHumans) <= 0  {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          return
        }
        human := dbHumans[0]

        if human != (idp.Human{}) {

          // FIXME: Use challenge system!

          challenge, err := idp.CreateDeleteChallenge(config.GetString("delete.link"), human, 60 * 5) // Fixme configfy 60*5
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            log.Debug(err.Error())
            return
          }

          hashedCode, err := idp.CreatePassword(challenge.Code)
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            log.Debug(err.Error())
            return
          }

          n := idp.Human{
            Identity: idp.Identity{
              Id: human.Id,
              OtpDeleteCode: hashedCode,
              OtpDeleteCodeExpire: challenge.Expire,
            },
          }
          updatedHuman, err := idp.UpdateOtpDeleteCode(tx, n)
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            log.Debug(err.Error())
            return
          }

          if updatedHuman != (idp.Human{}) {
            data := DeleteTemplateData{
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

            ok := client.DeleteHumansResponse{
              Id: updatedHuman.Id,
              RedirectTo: challenge.RedirectTo,
            }
            request.Output = bulky.NewOkResponse(request.Index, ok)
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
        log.Debug("Delete human failed. Hint: Maybe input validation needs to be improved.")
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
