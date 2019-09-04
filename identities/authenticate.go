package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  hydra "github.com/charmixer/hydra/client"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
)

type AuthenticateRequest struct {
  Id              string            `json:"id"`
  Password        string            `json:"password"`
  Challenge       string            `json:"challenge" binding:"required"`
  OtpChallenge    string            `json:"otp_challenge"`
}

type AuthenticateResponse struct {
  Id              string            `json:"id" binding:"required"`
  NotFound        bool              `json:"not_found" binding:"required"`
  Authenticated   bool              `json:"authenticated" binding:"required"`
  TotpRequired    bool              `json:"totp_required" binding:"required"`
  RedirectTo      string            `json:"redirect_to" binding:"required"`
}

func PostAuthenticate(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostAuthenticate",
    })

    var input AuthenticateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Create a new HTTP client to perform the request, to prevent serialization
    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    hydraLoginResponse, err := hydra.GetLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.login"), hydraClient, input.Challenge)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    log.WithFields(logrus.Fields{
      "challenge": input.Challenge,
      "skip": hydraLoginResponse.Skip,
      "redirect_to": hydraLoginResponse.RedirectTo,
      "subject": hydraLoginResponse.Subject,
    }).Debug("PostAuthenticate.Hydra.GetLogin.Response")

    if hydraLoginResponse.Skip {

      hydraLoginAcceptRequest := hydra.LoginAcceptRequest{
        Subject: hydraLoginResponse.Subject,
        Remember: true,
        RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
      }

      hydraLoginAcceptResponse := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, input.Challenge, hydraLoginAcceptRequest)
      log.WithFields(logrus.Fields{
        "challenge": input.Challenge,
        "redirect_to": hydraLoginAcceptResponse.RedirectTo,
      }).Debug("PostAuthenticate.Hydra.AcceptLogin.Response")

      acceptResponse := AuthenticateResponse{
        Id: hydraLoginResponse.Subject,
        Authenticated: true,
        NotFound: false,
        TotpRequired: false,
        RedirectTo: hydraLoginAcceptResponse.RedirectTo,
      }

      log.WithFields(logrus.Fields{
        "challenge": input.Challenge,
        "id": acceptResponse.Id,
        "authenticated": acceptResponse.Authenticated,
        "totp_required": acceptResponse.TotpRequired,
        "redirect_to": acceptResponse.RedirectTo,
      }).Debug("Authenticated")

      c.JSON(http.StatusOK, acceptResponse)
      c.Abort()
      return
    }

    denyResponse := AuthenticateResponse{
      Id: input.Id,
      NotFound: false,
      Authenticated: false,
      TotpRequired: false,
      RedirectTo: "",
    }

    // Masked read on challenge that has not been bound to an Identity yet. No need to hit database.
    if input.Challenge != "" && input.Id == "" {
      log.WithFields(logrus.Fields{
        "challenge": input.Challenge,
        "id": denyResponse.Id,
        "authenticated": denyResponse.Authenticated,
        "totp_required": denyResponse.TotpRequired,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
      c.Abort()
      return;
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

      otpChallenge, err := idp.FetchChallenge(env.Driver, input.OtpChallenge)
      if err != nil {
        log.Debug(err.Error())
        log.WithFields(logrus.Fields{
          "challenge": input.Challenge,
          "id": denyResponse.Id,
          "authenticated": denyResponse.Authenticated,
          "totp_required": denyResponse.TotpRequired,
          "redirect_to": denyResponse.RedirectTo,
        }).Debug("Authentication denied")
        c.JSON(http.StatusOK, denyResponse)
        c.Abort()
        return;
      }

      if otpChallenge == nil {
        log.WithFields(logrus.Fields{
          "challenge": input.Challenge,
          "id": denyResponse.Id,
          "authenticated": denyResponse.Authenticated,
          "totp_required": denyResponse.TotpRequired,
          "redirect_to": denyResponse.RedirectTo,
        }).Debug("Authentication denied")
        c.JSON(http.StatusOK, denyResponse)
        c.Abort()
        return;
      }

      if otpChallenge.Verified > 0 {

      }

      // Deny by default
      log.WithFields(logrus.Fields{
        "id": denyResponse.Id,
        "authenticated": denyResponse.Authenticated,
        "totp_required": denyResponse.TotpRequired,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
    }

    identities, err := idp.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      log.WithFields(logrus.Fields{
        "challenge": input.Challenge,
        "id": denyResponse.Id,
        "authenticated": denyResponse.Authenticated,
        "totp_required": denyResponse.TotpRequired,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
      c.Abort()
      return;
    }

    if identities != nil {

      log.WithFields(logrus.Fields{"fixme": 1}).Debug("Change FetchIdentitiesForSub to not be a bulk function")
      identity := identities[0];

      valid, _ := idp.ValidatePassword(identity.Password, input.Password)
      if valid == true {

        acceptResponse := AuthenticateResponse{
          Id: identity.Id,
          NotFound: false,
          Authenticated: true,
          TotpRequired: identity.TotpRequired,
          RedirectTo: "",
        }

        if identity.TotpRequired {
          // Do not call hydra yet we need totp authentication aswell. Create a totp request instaed.

          redirectTo := "https://id.localhost/authenticate?login_challenge=" + input.Challenge

          aChallenge := idp.Challenge{
            TTL: 300, // 5 min
            CodeType: "TOTP", // means the code is not stored in DB, but calculated from otp_secret
            Code: "",
            RedirectTo: redirectTo, // When challenge is verified where should the verify controller redirect to and append otp_challenge=
            Audience: "https://id.localhost/api/authenticate",
          }

          challenge, err := idp.CreateChallengeForIdentity(env.Driver, identity, aChallenge)
          if err != nil {
            log.Debug(err.Error())
            log.WithFields(logrus.Fields{
              "challenge": input.Challenge,
              "id": denyResponse.Id,
              "authenticated": denyResponse.Authenticated,
              "totp_required": denyResponse.TotpRequired,
              "redirect_to": denyResponse.RedirectTo,
            }).Debug("Authentication denied")
            c.JSON(http.StatusOK, denyResponse)
            c.Abort()
            return
          }

          // config.GetString("idpui.public.url") + config.GetString("idpui.public.url.endpoints.verify")
          acceptResponse.RedirectTo = "/verify?otp_challenge=" + challenge.OtpChallenge

        } else {
          hydraLoginAcceptRequest := hydra.LoginAcceptRequest{
            Subject: identity.Id,
            Remember: true,
            RememberFor: config.GetIntStrict("hydra.session.timeout"), // This means auto logout in hydra after n seconds!
          }
          hydraLoginAcceptResponse := hydra.AcceptLogin(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.loginAccept"), hydraClient, input.Challenge, hydraLoginAcceptRequest)
          log.WithFields(logrus.Fields{
            "challenge": input.Challenge,
            "redirect_to": hydraLoginAcceptResponse.RedirectTo,
          }).Debug("PostAuthenticate.Hydra.AcceptLogin.Response")
          acceptResponse.RedirectTo = hydraLoginAcceptResponse.RedirectTo
        }

        log.WithFields(logrus.Fields{
          "challenge": input.Challenge,
          "id": acceptResponse.Id,
          "authenticated": acceptResponse.Authenticated,
          "totp_required": acceptResponse.TotpRequired,
          "redirect_to": acceptResponse.RedirectTo,
        }).Debug("Authenticated")
        c.JSON(http.StatusOK, acceptResponse)
        return
      }

    } else {
      denyResponse.NotFound = true
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
    }

    // Deny by default
    log.WithFields(logrus.Fields{
      "id": denyResponse.Id,
      "authenticated": denyResponse.Authenticated,
      "totp_required": denyResponse.TotpRequired,
      "redirect_to": denyResponse.RedirectTo,
    }).Debug("Authentication denied")
    c.JSON(http.StatusOK, denyResponse)
  }
  return gin.HandlerFunc(fn)
}
