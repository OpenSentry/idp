package humans

import (
  "time"
  "net/url"
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  hydra "github.com/charmixer/hydra/client"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  "github.com/charmixer/idp/utils"
)

func PostAuthenticate(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostAuthenticate",
    })

    var requests []client.CreateIdentitiesAuthenticateRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    deny := client.IdentityAuthentication{}

    var handleRequest = func(iRequests []*utils.Request) {

      for _, request := range iRequests {
        r := request.Request.(client.CreateIdentitiesAuthenticateRequest)

        log = log.WithFields(logrus.Fields{"challenge": r.Challenge})

        hydraLoginResponse, err := hydra.GetLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.login"), hydraClient, r.Challenge)
        if err != nil {
          request.Response = utils.NewInternalErrorResponse(request.Index)
          log.Debug(err.Error())
          continue
        }

        // Skip if hydra dictated it.
        if hydraLoginResponse.Skip == true {

          hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydra.LoginAcceptRequest{
            Subject: hydraLoginResponse.Subject,
            Remember: true,
            RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
          })
          if err != nil {
            request.Response = utils.NewInternalErrorResponse(request.Index)
            log.Debug(err.Error())
            continue
          }

          log.WithFields(logrus.Fields{"fixme": 1}).Debug("Assert that the Identity found by Hydra still exists in IDP")
          accept := client.IdentityAuthentication{
            Id: hydraLoginResponse.Subject,
            Authenticated: true,
            RedirectTo: hydraLoginAcceptResponse.RedirectTo,
            TotpRequired: false,
            IsPasswordInvalid: false,
            IdentityExists: true, // FIXME: Check if Identity still exists in the system
          }

          var response client.CreateIdentitiesAuthenticateResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = accept
          request.Response = response

          log.WithFields(logrus.Fields{"skip":1, "id":accept.Id}).Debug("Authenticated")
          continue
        }

        // Check for OTP. Gets set when authenticated using password then redirected here after otp verification.
        if r.OtpChallenge != "" {

          // Cases:
          // 1. User needs to authenticate
          //   1. User passes, call accept login
          //   2. User passes, but require otp before accept login
          //     3. Generate otp challenge and redirect to verify
          //     4. User verifies and redirect back to authenticate on login_challenge but now with verified otp_challenge.

          // Check that login_challenge url and login_challenge in otp_challenge is the same. (CSRF)
          // Accept login based on verified otp data

          log = log.WithFields(logrus.Fields{
            "otp_challenge": r.OtpChallenge,
          })

          var otpChallenge []idp.Challenge
          otpChallenge = append(otpChallenge, idp.Challenge{Id: r.OtpChallenge})
          dbChallenges, err := idp.FetchChallenges(env.Driver, otpChallenge)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }
          if dbChallenges == nil {
            log.Debug("Challenge not found")
            request.Response = utils.NewClientErrorResponse(request.Index, []client.ErrorResponse{ {Code: -399 , Error:"Challenge not found"} })
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
              request.Response = utils.NewInternalErrorResponse(request.Index)
              continue
            }

            log.WithFields(logrus.Fields{"fixme": 1}).Debug("Assert that the Identity found by Hydra still exists in IDP")
            accept := client.IdentityAuthentication{
              Id: hydraLoginResponse.Subject,
              Authenticated: true,
              RedirectTo: hydraLoginAcceptResponse.RedirectTo,
              TotpRequired: false,
              IsPasswordInvalid: false,
              IdentityExists: true, // FIXME: Check if Identity still exists in the system
            }

            var response client.CreateIdentitiesAuthenticateResponse
            response.Index = request.Index
            response.Status = http.StatusOK
            response.Ok = accept
            request.Response = response

            log.WithFields(logrus.Fields{"acr":"totp", "id":accept.Id}).Debug("Authenticated")
            continue
          }

          // Deny by default
          var response client.CreateIdentitiesAuthenticateResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = deny
          request.Response = response
          log.WithFields(logrus.Fields{"2fa":"totp"}).Debug("Authentication denied")
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
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          if humans == nil {
            log.WithFields(logrus.Fields{"id": r.Id}).Debug("Identity not found")
            request.Response = utils.NewClientErrorResponse(request.Index, []client.ErrorResponse{ {Code: -380 , Error:"Identity not found"} })
            continue
          }
          human := humans[0]

          if human.AllowLogin == true {

            valid, _ := idp.ValidatePassword(human.Password, r.Password)
            if valid == true {

              accept := client.IdentityAuthentication{
                Id: human.Id,
                Authenticated: true,
                RedirectTo: "",
                TotpRequired: human.TotpRequired,
                IsPasswordInvalid: false,
                IdentityExists: true,
              }

              if human.TotpRequired == true {
                // Do not call hydra yet we need totp authentication aswell. Create a totp request instaed.

                log.WithFields(logrus.Fields{"fixme": 1}).Debug("Move idpui redirect into config")
                redirectTo := "https://id.localhost/login?login_challenge=" + r.Challenge

                newChallenge := idp.Challenge{
                  JwtRegisteredClaims: idp.JwtRegisteredClaims{
                    Issuer: config.GetString("idp.public.issuer"),
                    ExpiresAt: time.Now().Unix() + 300, // 5 min
                    Audience: "https://id.localhost/api/authenticate",
                  },
                  CodeType: "TOTP", // means the code is not stored in DB, but calculated from otp_secret
                  Code: "",
                  RedirectTo: redirectTo, // When challenge is verified where should the verify controller redirect to and append otp_challenge=
                }

                otpChallenge, _, err := idp.CreateChallenge(env.Driver, newChallenge, human.Id)
                if err != nil {
                  log.Debug(err.Error())
                  request.Response = utils.NewInternalErrorResponse(request.Index)
                  continue
                }

                log.WithFields(logrus.Fields{"fixme": 1}).Debug("IDP _MUST_ NOT HAVE BINDING TO IDPUI! - Find a way to make verify redirect setup")
                u, err := url.Parse( config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.verify") )
                if err != nil {
                  log.Debug(err.Error())
                  request.Response = utils.NewInternalErrorResponse(request.Index)
                  continue
                }
                q := u.Query()
                q.Add("otp_challenge", otpChallenge.Id)
                u.RawQuery = q.Encode()

                accept.RedirectTo = u.String()

              } else {
                hydraLoginAcceptRequest := hydra.LoginAcceptRequest{
                  Subject: human.Id,
                  Remember: true,
                  RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
                }
                hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, r.Challenge, hydraLoginAcceptRequest)
                if err != nil {
                  log.Debug(err.Error())
                  request.Response = utils.NewInternalErrorResponse(request.Index)
                  continue
                }

                accept.RedirectTo = hydraLoginAcceptResponse.RedirectTo
              }

              var response client.CreateIdentitiesAuthenticateResponse
              response.Index = request.Index
              response.Status = http.StatusOK
              response.Ok = accept
              request.Response = response

              log.WithFields(logrus.Fields{"id":accept.Id}).Debug("Authenticated")
              continue;
            } else {

              deny.IsPasswordInvalid = true

            }

          }

          // Deny by default
          log.WithFields(logrus.Fields{"id": r.Id}).Debug("Authentication denied")
          request.Response = deny
        }

      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{})

    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
