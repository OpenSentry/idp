package humans

import (
  "net/url"
  "net/http"
  "time"
  "context"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/opensentry/idp/app"
  "github.com/opensentry/idp/config"
  "github.com/opensentry/idp/gateway/idp"
  "github.com/opensentry/idp/client"
  E "github.com/opensentry/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
  hydra "github.com/charmixer/hydra/client"
)

type ConfirmTemplateData struct {
  Challenge string
  Id string
  Code string
  Sender string
  Email string
}

func PostAuthenticate(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

		ctx := context.TODO()

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostAuthenticate",
    })

    var requests []client.CreateHumansAuthenticateRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    controllerVerifyOtp := config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.verify")
    redirectToVerifyOtp, err := url.Parse(controllerVerifyOtp)
    if err != nil {
      log.WithFields(logrus.Fields{ "url":controllerVerifyOtp }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    controllerEmailConfirm := config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.emailconfirm")
    redirectToConfirmEmail, err := url.Parse(controllerEmailConfirm)
    if err != nil {
      log.WithFields(logrus.Fields{ "url":controllerEmailConfirm }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    controllerLogin := config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.login")
    redirectToLogin, err := url.Parse(controllerLogin)
    if err != nil {
      log.WithFields(logrus.Fields{ "url":controllerLogin }).Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
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

      tx, err := env.Driver.BeginTx(c, nil)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }

      for _, request := range iRequests {
        r := request.Input.(client.CreateHumansAuthenticateRequest)

        log = log.WithFields(logrus.Fields{"challenge": r.Challenge})

        hydraLoginResponse, err := hydra.GetLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.login"), hydraClient, r.Challenge)
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

        var human idp.Human
        var application idp.Client

        var subject string = hydraLoginResponse.Subject
        var clientId string = hydraLoginResponse.Client.ClientId

        deny := client.CreateHumansAuthenticateResponse{}
        deny.Id = subject

        // Lookup subject
        if subject != "" {
          humans, err := idp.FetchHumans(ctx, tx, []idp.Human{ {Identity: idp.Identity{Id:subject}} })
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
          if len(humans) <= 0 {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            deny.IdentityExists = false

            // Hydra told us subject has active session, but our identity provider told us that the subject no longer exists.
            // This happens if we delete identity from our db, but fail to revoke sessions from hydra.

            hydraLoginRejectResponse, err := hydra.RejectLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginReject"), hydraClient, r.Challenge, hydra.LoginRejectRequest{
              Error: "Identity deleted",
              ErrorDebug: "Identity " + r.Id + " does not exist in IDP, but still has active session. Forgot to revoke session when deleting Identity?",
              ErrorDescription: "Identity no longer exists, but still has active sessions",
              ErrorHint: "Restart the login process.",
              StatusCode: http.StatusUnauthorized,
            })
            if err != nil {
              log.Debug(err.Error())
              bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
              request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
              return
            }
            deny.RedirectTo = hydraLoginRejectResponse.RedirectTo

            // Revoke all sessions on subject in hydra, and reject login challenge.
            _, _ = hydra.DeleteLoginSessions(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.sessionsLogin"), hydraClient, hydra.DeleteLoginSessionRequest{Subject:subject})
            /*if err != nil {
              log.Debug(err.Error())
              bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
              request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
              return
            }
            log.Debug(deleteLoginSessionsResponse)*/

            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewOkResponse(request.Index, deny)
            return
          }
          human = humans[0]
        }

        // Lookup client
        if clientId != "" {
          clients, err := idp.FetchClients(ctx, tx, []idp.Client{ {Identity: idp.Identity{Id:clientId}} })
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
            log.WithFields(logrus.Fields{"id":clientId}).Debug("Client not found")
            return
          }
          if len(clients) <= 0 {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
            log.Debug(err.Error())
            return
          }
          application = clients[0]
        }

        // Skip if hydra dictated it.
        if hydraLoginResponse.Skip == true {
          acr := "skip"

          hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydra.LoginAcceptRequest{
            Subject: hydraLoginResponse.Subject,
            Remember: true,
            RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
            ACR: acr,
            Context: map[string]string{
              "client_name": application.Name,
              "subject_name": human.Name,
              "subject_email": human.Email,
            },
          })
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

          log.WithFields(logrus.Fields{"fixme": 1}).Debug("Assert that the Identity found by Hydra still exists in IDP")
          accept := client.CreateHumansAuthenticateResponse{
            Id: hydraLoginResponse.Subject,
            Authenticated: true,
            RedirectTo: hydraLoginAcceptResponse.RedirectTo,
            TotpRequired: false,
            IsPasswordInvalid: false,
            IdentityExists: true, // FIXME: Check if Identity still exists in the system
          }

          log.WithFields(logrus.Fields{"acr":acr, "id":accept.Id}).Debug("Authenticated")
          request.Output = bulky.NewOkResponse(request.Index, accept)
          idp.EmitEventIdentityAuthenticated(env.Nats, idp.Identity{Id: accept.Id}, acr)
          continue
        }

        // Check that email is confirmed before allowing login
        if r.EmailChallenge != "" {
          log = log.WithFields(logrus.Fields{ "email_challenge": r.EmailChallenge })

          acr := "otp.email"

          dbChallenges, err := idp.FetchChallenges(ctx, tx, []idp.Challenge{ {Id: r.EmailChallenge} })
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

          if len(dbChallenges) <= 0 {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
            return
          }

          challenge := dbChallenges[0]

          if challenge.VerifiedAt > 0 {
            _, err := idp.ConfirmEmail(ctx, tx, idp.Human{ Identity: idp.Identity{Id: challenge.Subject} })
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
            log.WithFields(logrus.Fields{"id":challenge.Subject}).Debug("Email Confirmed")

            log.WithFields(logrus.Fields{"fixme": 1}).Debug("Check if challenge actually matches login_challenge and that session matches?")

            hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydra.LoginAcceptRequest{
              Subject: challenge.Subject,
              Remember: true,
              RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
              ACR: acr,
              Context: map[string]string{
                "client_name": application.Name,
                "subject_name": human.Name,
                "subject_email": human.Email,
              },
            })
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

            log.WithFields(logrus.Fields{"fixme": 1}).Debug("Assert that the Identity found by Hydra still exists in IDP")
            accept := client.CreateHumansAuthenticateResponse{
              Id: challenge.Subject,
              Authenticated: true,
              RedirectTo: hydraLoginAcceptResponse.RedirectTo,
              TotpRequired: false,
              IsPasswordInvalid: false,
              IdentityExists: true, // FIXME: Check if Identity still exists in the system
            }

            log.WithFields(logrus.Fields{"acr":acr, "id":accept.Id}).Debug("Authenticated")
            request.Output = bulky.NewOkResponse(request.Index, accept)
            idp.EmitEventIdentityAuthenticated(env.Nats, idp.Identity{Id: accept.Id}, acr)
            continue
          }

        }

        // Check for OTP. Gets set when authenticated using password then redirected here after otp verification.
        if r.OtpChallenge != "" {
          log = log.WithFields(logrus.Fields{ "otp_challenge": r.OtpChallenge })
          acr := "otp"

          dbChallenges, err := idp.FetchChallenges(ctx, tx, []idp.Challenge{ {Id: r.OtpChallenge} })
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

          if len(dbChallenges) <= 0 {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
            return
          }

          challenge := dbChallenges[0]

          if challenge.VerifiedAt > 0 {
            log.WithFields(logrus.Fields{"id":challenge.Subject}).Debug("OTP Verified")

            log.WithFields(logrus.Fields{"fixme": 1}).Debug("Check if challenge actually matches login_challenge and that session matches?")

            hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydra.LoginAcceptRequest{
              Subject: challenge.Subject,
              Remember: true,
              RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
              ACR: acr,
              Context: map[string]string{
                "client_name": application.Name,
                "subject_name": human.Name,
                "subject_email": human.Email,
              },
            })
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

            log.WithFields(logrus.Fields{"fixme": 1}).Debug("Assert that the Identity found by Hydra still exists in IDP")
            accept := client.CreateHumansAuthenticateResponse{
              Id: challenge.Subject,
              Authenticated: true,
              RedirectTo: hydraLoginAcceptResponse.RedirectTo,
              TotpRequired: false,
              IsPasswordInvalid: false,
              IdentityExists: true, // FIXME: Check if Identity still exists in the system
            }

            log.WithFields(logrus.Fields{"acr":acr, "id":accept.Id}).Debug("Authenticated")
            request.Output = bulky.NewOkResponse(request.Index, accept)
            idp.EmitEventIdentityAuthenticated(env.Nats, idp.Identity{Id: accept.Id}, acr)
            continue
          }

          // Deny by default
          log.WithFields(logrus.Fields{"acr":acr}).Debug("Authentication denied")
          request.Output = bulky.NewOkResponse(request.Index, deny)
          continue
        }

        /*
        // Masked read on challenge that has not been bound to an Identity yet. No need to hit database.
        if input.Challenge != "" && input.Id == "" {
          logResponse(log, input.Challenge, denyResponse, "Authentication denied")
          c.JSON(http.StatusOK, denyResponse)
          return;
        }
        */

        if r.Id != "" {

          log = log.WithFields(logrus.Fields{"id": r.Id})

          dbHumans, err := idp.FetchHumans(ctx, tx, []idp.Human{ {Identity:idp.Identity{Id:r.Id}} })
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

          if len(dbHumans) <= 0 {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
            return
          }
          human := dbHumans[0]

          deny.Id = human.Id

          if human.AllowLogin == true {

            valid, _ := idp.ValidatePassword(human.Password, r.Password)
            if valid == true {

              accept := client.CreateHumansAuthenticateResponse{
                Id: human.Id,
                Authenticated: true,
                RedirectTo: "",
                TotpRequired: human.TotpRequired,
                IsPasswordInvalid: false,
                IdentityExists: true,
              }

              // When challenge is verified where should the controller redirect to and append its challenge
              redirectToUrlWhenVerified := redirectToLogin
              q := redirectToUrlWhenVerified.Query()
              q.Add("login_challenge", r.Challenge)
              redirectToUrlWhenVerified.RawQuery = q.Encode()

              if human.EmailConfirmedAt <= 0 || human.TotpRequired == true {

                if human.EmailConfirmedAt <= 0 {

                  // Require email confirmation challenge

                  newChallenge := idp.Challenge{
                    JwtRegisteredClaims: idp.JwtRegisteredClaims{
                      Subject: human.Id,
                      Issuer: config.GetString("idp.public.issuer"),
                      Audience: config.GetString("idp.public.url") + config.GetString("idp.public.endpoints.challenges.verify"),
                      ExpiresAt: time.Now().Unix() + 900, // 15 min,  FIXME: Should be configurable
                    },
                    RedirectTo: redirectToUrlWhenVerified.String(),
                    CodeType: int64(client.OTP),
                    Data: human.Email,
                  }
                  challenge, otpCode, err := idp.CreateChallengeUsingOtp(ctx, tx, idp.ChallengeAuthenticate, newChallenge)
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

                    if otpCode.Code != "" && human.Email != "" {

                      var data = ConfirmTemplateData{
                        Challenge: challenge.Id,
                        Sender: sender.Name,
                        Id: challenge.Subject,
                        Email: human.Email,
                        Code: otpCode.Code, // Note this is the clear text generated code and not the hashed one stored in DB.
                      }
                      _, err = idp.SendEmailUsingTemplate(smtpConfig, human.Email, human.Email, emailSubject, templateFile, data)
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

                    q = redirectToConfirmEmail.Query()
                    q.Add("email_challenge", challenge.Id)
                    redirectToConfirmEmail.RawQuery = q.Encode()

                    accept.RedirectTo = redirectToConfirmEmail.String()
                  }

                } else if human.TotpRequired == true {

                  // Require totp challenge

                  newChallenge := idp.Challenge{
                    JwtRegisteredClaims: idp.JwtRegisteredClaims{
                      Subject: human.Id,
                      Issuer: config.GetString("idp.public.issuer"),
                      Audience: config.GetString("idp.public.url") + config.GetString("idp.public.endpoints.challenges.verify"),
                      ExpiresAt: time.Now().Unix() + 300, // 5 min, FIXME: Should be configurable
                    },
                    RedirectTo: redirectToUrlWhenVerified.String(),
                    CodeType: int64(client.TOTP),
                  }
                  challenge, err := idp.CreateChallengeUsingTotp(ctx, tx, idp.ChallengeAuthenticate, newChallenge)
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

                  q = redirectToVerifyOtp.Query()
                  q.Add("otp_challenge", challenge.Id)
                  redirectToVerifyOtp.RawQuery = q.Encode()

                  accept.RedirectTo = redirectToVerifyOtp.String()
                }

              } else {

                // All verification requirements completed, so call accept in hydra.
                acr := ""
                hydraLoginAcceptRequest := hydra.LoginAcceptRequest{
                  Subject: human.Id,
                  Remember: true,
                  RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
                  ACR: acr,
                  Context: map[string]string{
                    "client_name": application.Name,
                    "subject_name": human.Name,
                    "subject_email": human.Email,
                  },
                }
                hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydraLoginAcceptRequest)
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

                accept.RedirectTo = hydraLoginAcceptResponse.RedirectTo

                log.WithFields(logrus.Fields{"id":accept.Id, "acr":acr}).Debug("Authenticated")
                idp.EmitEventIdentityAuthenticated(env.Nats, idp.Identity{Id: accept.Id}, acr)
              }

              request.Output = bulky.NewOkResponse(request.Index, accept)
              continue

            } else {

              deny.IsPasswordInvalid = true

            }

          }

        }

        // Deny by default
        log.WithFields(logrus.Fields{"id": r.Id}).Debug("Authentication denied")
        request.Output = bulky.NewOkResponse(request.Index, deny)
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
