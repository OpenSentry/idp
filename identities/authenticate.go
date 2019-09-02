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
  Passcode        string            `json:"passcode"`
  Challenge       string            `json:"challenge" binding:"required"`
}

type AuthenticateResponse struct {
  Id              string            `json:"id" binding:"required"`
  NotFound        bool              `json:"not_found" binding:"required"`
  Authenticated   bool              `json:"authenticated" binding:"required"`
  Require2Fa      bool              `json:"require_2fa" binding:"required"`
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

    log.WithFields(logrus.Fields{
      "challenge": input.Challenge,
      "id": input.Id,
      "password": input.Password,
    }).Debug("PostAuthenticate.Input")

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

      log.WithFields(logrus.Fields{"fixme":1}).Debug("Should we ignore 2fa require when hydra says skip?")
      acceptResponse := AuthenticateResponse{
        Id: hydraLoginResponse.Subject,
        Authenticated: true,
        NotFound: false,
        Require2Fa: false,
        RedirectTo: hydraLoginAcceptResponse.RedirectTo,
      }

      log.WithFields(logrus.Fields{
        "challenge": input.Challenge,
        "id": acceptResponse.Id,
        "authenticated": acceptResponse.Authenticated,
        "require_2fa": acceptResponse.Require2Fa,
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
      Require2Fa: false,
      RedirectTo: "",
    }

    // Masked read on challenge, no need to hit database.
    if input.Challenge != "" && input.Id == "" {
      log.WithFields(logrus.Fields{
        "challenge": input.Challenge,
        "id": denyResponse.Id,
        "authenticated": denyResponse.Authenticated,
        "require_2fa": denyResponse.Require2Fa,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Authentication denied")
      c.JSON(http.StatusOK, denyResponse)
      c.Abort()
      return;
    }

    identities, err := idp.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      log.WithFields(logrus.Fields{
        "challenge": input.Challenge,
        "id": denyResponse.Id,
        "authenticated": denyResponse.Authenticated,
        "require_2fa": denyResponse.Require2Fa,
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
          Require2Fa: identity.Require2Fa,
          RedirectTo: "",
        }

        if identity.Require2Fa {
          // Do not call hydra yet we need passcode authentication aswell. Create a passcode request instaed.
          log.WithFields(logrus.Fields{"fixme": 1}).Debug("How to do the passcode redirect in config? Maybe make request param a jwt");
          url := "/passcode" //config.GetString("idpui.public.url") + config.GetString("idpui.public.url.endpoints.passcode")
          passcodeChallenge := idp.CreatePasscodeChallenge(url, input.Challenge, identity.Id, config.GetString("2fa.sigkey"))
          acceptResponse.RedirectTo = passcodeChallenge.RedirectTo

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
          "require_2fa": acceptResponse.Require2Fa,
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
      "require_2fa": denyResponse.Require2Fa,
      "redirect_to": denyResponse.RedirectTo,
    }).Debug("Authentication denied")
    c.JSON(http.StatusOK, denyResponse)
  }
  return gin.HandlerFunc(fn)
}
