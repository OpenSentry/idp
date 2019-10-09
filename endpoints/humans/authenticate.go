package humans

import (
  "time"
  "net/url"
  "net/http"
  "text/template"
  "io/ioutil"
  "bytes"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"  
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
  hydra "github.com/charmixer/hydra/client"
)

type EmailConfirmTemplateData struct {
  Name string
  Code string
  Sender string
}

func PostAuthenticate(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
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

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.CreateHumansAuthenticateRequest)

        log = log.WithFields(logrus.Fields{"challenge": r.Challenge})

        hydraLoginResponse, err := hydra.GetLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.login"), hydraClient, r.Challenge)
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        deny := client.CreateHumansAuthenticateResponse{}
        deny.Id = hydraLoginResponse.Subject

        // Skip if hydra dictated it.
        if hydraLoginResponse.Skip == true {

          hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydra.LoginAcceptRequest{
            Subject: hydraLoginResponse.Subject,
            Remember: true,
            RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
          })
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
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

          log.WithFields(logrus.Fields{"skip":1, "id":accept.Id}).Debug("Authenticated")
          request.Output = bulky.NewOkResponse(request.Index, accept)
          continue
        }

        // Check that email is confirmed before allowing login
        if r.EmailChallenge != "" {
          log = log.WithFields(logrus.Fields{ "email_challenge": r.EmailChallenge })

          dbChallenges, err := idp.FetchChallenges(env.Driver, []idp.Challenge{ {Id: r.EmailChallenge} })
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }
          if dbChallenges == nil {
            log.Debug("Challenge not found")
            request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
            continue
          }
          challenge := dbChallenges[0]

          if challenge.VerifiedAt > 0 {
            _, err := idp.ConfirmEmail(env.Driver, idp.Human{ Identity: idp.Identity{Id: challenge.Subject} })
            if err != nil {
              log.Debug(err.Error())
              request.Output = bulky.NewInternalErrorResponse(request.Index)
              continue
            }
            log.WithFields(logrus.Fields{"id":challenge.Subject}).Debug("Email Confirmed")

            log.WithFields(logrus.Fields{"fixme": 1}).Debug("Check if challenge actually matches login_challenge and that session matches?")

            hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydra.LoginAcceptRequest{
              Subject: challenge.Subject,
              Remember: true,
              RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
            })
            if err != nil {
              log.Debug(err.Error())
              request.Output = bulky.NewInternalErrorResponse(request.Index)
              continue
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

            log.WithFields(logrus.Fields{"acr":"otp,email", "id":accept.Id}).Debug("Authenticated")
            request.Output = bulky.NewOkResponse(request.Index, accept)
            continue
          }

        }

        // Check for OTP. Gets set when authenticated using password then redirected here after otp verification.
        if r.OtpChallenge != "" {
          log = log.WithFields(logrus.Fields{ "otp_challenge": r.OtpChallenge })

          dbChallenges, err := idp.FetchChallenges(env.Driver, []idp.Challenge{ {Id: r.OtpChallenge} })
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }
          if dbChallenges == nil {
            log.Debug("Challenge not found")
            request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
            continue
          }
          challenge := dbChallenges[0]

          if challenge.VerifiedAt > 0 {
            log.WithFields(logrus.Fields{"id":challenge.Subject}).Debug("OTP Verified")

            log.WithFields(logrus.Fields{"fixme": 1}).Debug("Check if challenge actually matches login_challenge and that session matches?")

            hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydra.LoginAcceptRequest{
              Subject: challenge.Subject,
              Remember: true,
              RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
            })
            if err != nil {
              log.Debug(err.Error())
              request.Output = bulky.NewInternalErrorResponse(request.Index)
              continue
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

            log.WithFields(logrus.Fields{"acr":"totp", "id":accept.Id}).Debug("Authenticated")
            request.Output = bulky.NewOkResponse(request.Index, accept)
            continue
          }

          // Deny by default
          log.WithFields(logrus.Fields{"2fa":"totp"}).Debug("Authentication denied")
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

          humans, err := idp.FetchHumansById( env.Driver, []string{r.Id} )
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          if humans == nil {
            log.WithFields(logrus.Fields{"id": r.Id}).Debug("Human not found")
            request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
            continue
          }
          human := humans[0]

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

              redirectToUrlWhenVerified, err := url.Parse( config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.login") )
              if err != nil {
                log.Debug(err.Error())
                request.Output = bulky.NewInternalErrorResponse(request.Index)
                continue
              }
              // When challenge is verified where should the controller redirect to and append its challenge
              q := redirectToUrlWhenVerified.Query()
              q.Add("login_challenge", r.Challenge)
              redirectToUrlWhenVerified.RawQuery = q.Encode()

              if human.EmailConfirmedAt <= 0 || human.TotpRequired == true {

                var err error
                var challengeKey string
                var redirectToConfirm *url.URL
                var challenge idp.Challenge
                var newChallenge idp.Challenge
                var sendChallengeEmail string
                var challengeCode string

                if human.EmailConfirmedAt <= 0 {

                  challengeKey = "email_challenge"
                  sendChallengeEmail = human.Email

                  epVerifyController := config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.emailconfirm")
                  redirectToConfirm, err = url.Parse(epVerifyController)
                  if err != nil {
                    log.WithFields(logrus.Fields{ "url":epVerifyController }).Debug(err.Error())
                    request.Output = bulky.NewInternalErrorResponse(request.Index)
                    continue
                  }

                  emailChallenge, err := idp.CreateChallengeCode()
                  if err != nil {
                    log.Debug(err.Error())
                    request.Output = bulky.NewInternalErrorResponse(request.Index)
                    continue
                  }
                  challengeCode = emailChallenge.Code

                  hashedCode, err := idp.CreatePassword(challengeCode)
                  if err != nil {
                    log.Debug(err.Error())
                    request.Output = bulky.NewInternalErrorResponse(request.Index)
                    continue
                  }

                  newChallenge = idp.Challenge{
                    JwtRegisteredClaims: idp.JwtRegisteredClaims{
                      Issuer: config.GetString("idp.public.issuer"),
                      ExpiresAt: time.Now().Unix() + 900, // 15 min
                      Audience: "https://id.localhost/api/challenges/verify",
                    },
                    CodeType: "OTP",
                    Code: hashedCode,
                    RedirectTo: redirectToUrlWhenVerified.String(),
                  }

                } else if human.TotpRequired == true {

                  challengeKey = "otp_challenge"

                  // Do not call hydra yet we need totp authentication aswell. Create a totp request instaed.
                  epVerifyController := config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.verify")
                  redirectToConfirm, err = url.Parse(epVerifyController)
                  if err != nil {
                    log.WithFields(logrus.Fields{ "url":epVerifyController }).Debug(err.Error())
                    request.Output = bulky.NewInternalErrorResponse(request.Index)
                    continue
                  }

                  newChallenge = idp.Challenge{
                    JwtRegisteredClaims: idp.JwtRegisteredClaims{
                      Issuer: config.GetString("idp.public.issuer"),
                      ExpiresAt: time.Now().Unix() + 300, // 5 min
                      Audience: "https://id.localhost/api/authenticate",
                    },
                    CodeType: "TOTP", // means the code is not stored in DB, but calculated from otp_secret
                    Code: "",
                    RedirectTo: redirectToUrlWhenVerified.String(),
                  }

                }

                challenge, _, err = idp.CreateChallenge(env.Driver, newChallenge, human.Id)
                if err != nil {
                  log.Debug(err.Error())
                  request.Output = bulky.NewInternalErrorResponse(request.Index)
                  continue
                }

                if sendChallengeEmail != "" {

                  log.WithFields(logrus.Fields{ "code":challengeCode, "email_challenge":challenge.Id }).Debug("EMAIL CONFIRMATION CODE - Do not log this in production!!!")

                  sender := idp.SMTPSender{ Name: config.GetString("emailconfirm.sender.name"), Email: config.GetString("emailconfirm.sender.email") }
                  templateFile := config.GetString("emailconfirm.template.email.file")
                  emailSubject := config.GetString("emailconfirm.template.email.subject")
                  _, err = sendChallengeToEmail(challenge, human.Email, human.Name, sender, templateFile, emailSubject, EmailConfirmTemplateData{
                    Sender: sender.Name,
                    Name: human.Name,
                    Code: challengeCode, // Note this is the clear text generated code and not the hashed one stored in DB.
                  })
                  if err != nil {
                    log.Debug(err.Error())
                    request.Output = bulky.NewInternalErrorResponse(request.Index)
                    continue
                  }
                }

                q = redirectToConfirm.Query()
                q.Add(challengeKey, challenge.Id)
                redirectToConfirm.RawQuery = q.Encode()

                accept.RedirectTo = redirectToConfirm.String()

              } else {

                // All verification requirements completed, so call accept in hydra.

                hydraLoginAcceptRequest := hydra.LoginAcceptRequest{
                  Subject: human.Id,
                  Remember: true,
                  RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
                }
                hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydraLoginAcceptRequest)
                if err != nil {
                  log.Debug(err.Error())
                  request.Output = bulky.NewInternalErrorResponse(request.Index)
                  continue
                }

                accept.RedirectTo = hydraLoginAcceptResponse.RedirectTo
              }

              log.WithFields(logrus.Fields{"id":accept.Id}).Debug("Authenticated")
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
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func sendChallengeToEmail(challenge idp.Challenge, email string, name string, sender idp.SMTPSender, templateFile string, emailSubject string, data interface{}) (bool, error) {

  smtpConfig := idp.SMTPConfig{
    Host: config.GetString("mail.smtp.host"),
    Username: config.GetString("mail.smtp.user"),
    Password: config.GetString("mail.smtp.password"),
    Sender: sender,
    SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
  }

  tplRecover, err := ioutil.ReadFile(templateFile)
  if err != nil {
    return false, err
  }

  t := template.Must(template.New(templateFile).Parse(string(tplRecover)))

  var tpl bytes.Buffer
  if err := t.Execute(&tpl, data); err != nil {
    return false, err
  }

  anEmail := idp.AnEmail{ Subject: emailSubject, Body: tpl.String()}

  _, err = idp.SendAnEmailToAnonymous(smtpConfig, name, email, anEmail)
  if err != nil {
    return false, err
  }

  return true, nil
}
