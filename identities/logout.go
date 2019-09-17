package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  hydra "github.com/charmixer/hydra/client"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  . "github.com/charmixer/idp/client"
)

func PostLogout(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostLogout",
    })

    var input IdentitiesLogoutRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Create a new HTTP client to perform the request, to prevent serialization
    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    hydraLogoutAcceptRequest := hydra.LogoutAcceptRequest{
    }
    hydraLogoutAcceptResponse, err := hydra.AcceptLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logoutAccept"), hydraClient, input.Challenge, hydraLogoutAcceptRequest)
    if err != nil {
      log.Fatal(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    logoutResponse := IdentitiesLogoutResponse{
      RedirectTo: hydraLogoutAcceptResponse.RedirectTo,
    }

    log.Debug("redirect_to: " + logoutResponse.RedirectTo)
    c.JSON(http.StatusOK, logoutResponse)
  }
  return gin.HandlerFunc(fn)
}
