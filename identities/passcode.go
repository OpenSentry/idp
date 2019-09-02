package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "github.com/CharMixer/hydra-client" // FIXME: Do not use upper case
  "idp/config"
  "idp/environment"
  "idp/gateway/idp"
)

type PasscodeRequest struct {
  Id        string `json:"id" binding:"required"`
  Passcode  string `json:"passcode" binding:"required"`
  Challenge string `json:"challenge" binding:"required"`
}

type PasscodeResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verified" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

func PostPasscode(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostPasscode",
    })

    var input PasscodeRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    denyResponse := PasscodeResponse{
      Id: input.Id,
      Verified: false,
      RedirectTo: "",
    }

    identities, err := idp.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      log.WithFields(logrus.Fields{
        "id": denyResponse.Id,
        "verified": denyResponse.Verified,
        "redirect_to": denyResponse.RedirectTo,
      }).Debug("Passcode rejected")
      c.JSON(http.StatusOK, denyResponse)
      c.Abort()
      return
    }

    if identities != nil {

      log.WithFields(logrus.Fields{"fixme": 1}).Debug("Change FetchIdentitiesForSub to not be a bulk function")
      identity := identities[0];

      // Sanity check. Only check password when 2fa is required by identity.
      if identity.Require2Fa == false {
        log.WithFields(logrus.Fields{
          "id": denyResponse.Id,
          "verified": denyResponse.Verified,
          "redirect_to": denyResponse.RedirectTo,
        }).Debug("Passcode rejected. Hint: Identity does not require 2fa")
        c.JSON(http.StatusOK, denyResponse)
        c.Abort()
        return
      }

      decryptedSecret, err := idp.Decrypt(identity.Secret2Fa, config.GetString("2fa.cryptkey"))
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      valid, _ := idp.ValidatePasscode(input.Passcode, decryptedSecret)
      if valid == true {

        hydraClient := hydra.NewHydraClient(env.HydraConfig)

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

        acceptResponse := PasscodeResponse{
          Id: identity.Id,
          Verified: true,
          RedirectTo: hydraLoginAcceptResponse.RedirectTo,
        }

        log.WithFields(logrus.Fields{
          "id": acceptResponse.Id,
          "verified": acceptResponse.Verified,
          "redirect_to": acceptResponse.RedirectTo,
        }).Debug("Passcode verified")

        c.JSON(http.StatusOK, acceptResponse)
        return
      }

    } else {
      log.WithFields(logrus.Fields{"id": input.Id}).Debug("Identity not found")
    }

    // Deny by default
    log.WithFields(logrus.Fields{
      "id": denyResponse.Id,
      "verified": denyResponse.Verified,
      "redirect_to": denyResponse.RedirectTo,
    }).Debug("Passcode rejected")
    c.JSON(http.StatusOK, denyResponse)
  }
  return gin.HandlerFunc(fn)
}
