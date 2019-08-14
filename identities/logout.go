package identities

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "github.com/CharMixer/hydra-client" // FIXME: Do not use upper case
  "golang-idp-be/config"
  "golang-idp-be/environment"
)

type LogoutRequest struct {
  Challenge       string            `json:"challenge" binding:"required"`
}

type LogoutResponse struct {
}

func PostLogout(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "route.logid": route.LogId,
      "component": "identities",
      "func": "PostLogout",
    })

    var input LogoutRequest
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

    log.Debug("redirect_to: " + hydraLogoutAcceptResponse.RedirectTo)
    c.JSON(http.StatusOK, gin.H{
      "redirect_to": hydraLogoutAcceptResponse.RedirectTo,
    })
  }
  return gin.HandlerFunc(fn)
}
