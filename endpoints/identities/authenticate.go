package identities

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
  . "github.com/charmixer/idp/client"
)

func PostAuthenticate(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostAuthenticate",
    })

    var input IdentitiesAuthenticateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    denyResponse := IdentitiesAuthenticateResponse{
      Id: input.Id,
      Authenticated: false,
      RedirectTo: "",
      TotpRequired: false,
      IsPasswordInvalid: false,
      IdentityExists: false,
    }

    // Create a new HTTP client to perform the request, to prevent serialization
    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    hydraLoginResponse, err := hydra.GetLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.login"), hydraClient, input.Challenge)
    if err != nil {
      log.Debug(err.Error())
      logResponse(log, input.Challenge, denyResponse, "Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
      return
    }
    log.WithFields(logrus.Fields{
      "challenge": input.Challenge,
      "skip": hydraLoginResponse.Skip,
      "redirect_to": hydraLoginResponse.RedirectTo,
      "subject": hydraLoginResponse.Subject,
    }).Debug("PostAuthenticate.Hydra.GetLogin.Response")

    if hydraLoginResponse.Skip == true {

      hydraLoginAcceptRequest := hydra.LoginAcceptRequest{
        Subject: hydraLoginResponse.Subject,
        Remember: true,
        RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
      }

      hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, input.Challenge, hydraLoginAcceptRequest)
      if err != nil {
        log.Debug(err.Error())
        logResponse(log, input.Challenge, denyResponse, "Authentication denied")
        c.JSON(http.StatusOK, denyResponse)
        return
      }
      log.WithFields(logrus.Fields{
        "challenge": input.Challenge,
        "redirect_to": hydraLoginAcceptResponse.RedirectTo,
      }).Debug("PostAuthenticate.Hydra.AcceptLogin.Response")

      log.WithFields(logrus.Fields{"fixme": 1}).Debug("Test if identity still exists in the system")
      acceptResponse := IdentitiesAuthenticateResponse{
        Id: hydraLoginResponse.Subject,
        Authenticated: true,
        RedirectTo: hydraLoginAcceptResponse.RedirectTo,
        TotpRequired: false,
        IsPasswordInvalid: false,
        IdentityExists: true,
      }

      logResponse(log, input.Challenge, acceptResponse, "Authenticated")
      c.JSON(http.StatusOK, acceptResponse)
      return
    }

    // Found otp_challenge check if verified.
    if input.OtpChallenge != "" {

      // Cases:
      // 1. User needs to authenticate
      //   1. User passes, call accept login
      //   2. User passes, but require otp before accept login
      //     3. Generate otp challenge and redirect to verify
      //     4. User verifies and redirect back to authenticate on login_challenge but now with verified otp_challenge.

      // Check that login_challenge url and login_challenge in otp_challenge is the same. (CSRF)
      // Accept login based on verified otp data

      challenges, err := idp.FetchChallengesById(env.Driver, []string{input.OtpChallenge})
      if err != nil {
        log.Debug(err.Error())
        logResponse(log, input.Challenge, denyResponse, "Authentication denied")
        c.JSON(http.StatusOK, denyResponse)
        return
      }

      if challenges == nil {
        log.WithFields(logrus.Fields{
          "otp_challenge": input.OtpChallenge,
        }).Debug("OTP Challenge not found")
        logResponse(log, input.Challenge, denyResponse, "Authentication denied")
        c.JSON(http.StatusOK, denyResponse)
        return
      }
      challenge := challenges[0]

      if challenge.VerifiedAt > 0 {
        log.WithFields(logrus.Fields{"otp_challenge": challenge.Id}).Debug("Challenge verified")

        log.WithFields(logrus.Fields{"fixme": 1}).Debug("Check if challenge actually matches login_challenge and that session matches?")

        hydraLoginAcceptRequest := hydra.LoginAcceptRequest{
          Subject: challenge.Subject,
          Remember: true,
          RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
        }
        hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, input.Challenge, hydraLoginAcceptRequest)
        if err != nil {
          log.Debug(err.Error())
          c.AbortWithStatus(http.StatusInternalServerError)
          return
        }
        log.WithFields(logrus.Fields{
          "challenge": input.Challenge,
          "redirect_to": hydraLoginAcceptResponse.RedirectTo,
        }).Debug("PostAuthenticate.Hydra.AcceptLogin.Response")

        log.WithFields(logrus.Fields{"fixme": 1}).Debug("Test if identity still exists in the system and if totp is still required")
        acceptResponse := IdentitiesAuthenticateResponse{
          Id: hydraLoginResponse.Subject,
          Authenticated: true,
          RedirectTo: hydraLoginAcceptResponse.RedirectTo,
          TotpRequired: true,
          IsPasswordInvalid: false,
          IdentityExists: true,
        }

        logResponse(log, input.Challenge, acceptResponse, "Authenticated")
        c.JSON(http.StatusOK, acceptResponse)
        return
      }

      // Deny by default
      logResponse(log, input.Challenge, denyResponse, "Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
      return
    }

    // Masked read on challenge that has not been bound to an Identity yet. No need to hit database.
    if input.Challenge != "" && input.Id == "" {
      logResponse(log, input.Challenge, denyResponse, "Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
      return;
    }

    humans, err := idp.FetchHumansById( env.Driver, []string{input.Id} )
    if err != nil {
      log.Debug(err.Error())
      logResponse(log, input.Challenge, denyResponse, "Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
      return;
    }

    if humans == nil {
      denyResponse.IdentityExists = false
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
      logResponse(log, input.Challenge, denyResponse, "Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
      return
    }
    human := humans[0]

    if human.AllowLogin == true {

      valid, _ := idp.ValidatePassword(human.Password, input.Password)
      if valid == true {

        acceptResponse := IdentitiesAuthenticateResponse{
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
          redirectTo := "https://id.localhost/login?login_challenge=" + input.Challenge

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
            logResponse(log, input.Challenge, denyResponse, "Authentication denied")
            c.JSON(http.StatusOK, denyResponse)
            return
          }

          log.WithFields(logrus.Fields{"fixme": 1}).Debug("IDP _MUST_ NOT HAVE BINDING TO IDPUI! - Find a way to make verify redirect setup")
          u, err := url.Parse( config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.verify") )
          if err != nil {
            log.Debug(err.Error())
            logResponse(log, input.Challenge, denyResponse, "Authentication denied")
            c.JSON(http.StatusOK, denyResponse)
            return
          }
          q := u.Query()
          q.Add("otp_challenge", otpChallenge.Id)
          u.RawQuery = q.Encode()

          acceptResponse.RedirectTo = u.String()

        } else {
          hydraLoginAcceptRequest := hydra.LoginAcceptRequest{
            Subject: human.Id,
            Remember: true,
            RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
          }
          hydraLoginAcceptResponse, err := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, input.Challenge, hydraLoginAcceptRequest)
          if err != nil {
            log.Debug(err.Error())
            c.AbortWithStatus(http.StatusInternalServerError)
            return
          }
          log.WithFields(logrus.Fields{
            "challenge": input.Challenge,
            "redirect_to": hydraLoginAcceptResponse.RedirectTo,
          }).Debug("PostAuthenticate.Hydra.AcceptLogin.Response")

          acceptResponse.RedirectTo = hydraLoginAcceptResponse.RedirectTo
        }

        logResponse(log, input.Challenge, acceptResponse, "Authenticated")
        c.JSON(http.StatusOK, acceptResponse)
        return
      } else {

        denyResponse.IsPasswordInvalid = true

      }

    }

    // Deny by default
    logResponse(log, input.Challenge, denyResponse, "Authentication denied")
    c.JSON(http.StatusOK, denyResponse)
  }
  return gin.HandlerFunc(fn)
}

func logResponse(log *logrus.Entry, challenge string, response IdentitiesAuthenticateResponse, msg string) {
  log.WithFields(logrus.Fields{
    "challenge": challenge,
    "id": response.Id,
    "authenticated": response.Authenticated,
    "redirect_to": response.RedirectTo,
    "totp_required": response.TotpRequired,
    "is_password_invalid": response.IsPasswordInvalid,
    "identity_exists": response.IdentityExists,
  }).Debug(msg)
}
