package identities

import (
  "net/http"

  "github.com/gin-gonic/gin"

  "golang-idp-be/config"
  "golang-idp-be/environment"
  //"golang-idp-be/gateway/idpbe"
  "golang-idp-be/gateway/hydra"
)

type LogoutRequest struct {
  Challenge       string            `json:"challenge" binding:"required"`
}

type LogoutResponse struct {
}

func PostLogout(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    requestId := c.MustGet("RequestId").(string)
    environment.DebugLog(route.LogId, "PostLogout", "", requestId)

    var input LogoutRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Create a new HTTP client to perform the request, to prevent serialization
    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    hydraLogoutAcceptRequest := hydra.HydraLogoutAcceptRequest{
    }
    hydraLogoutAcceptResponse, err := hydra.AcceptLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logoutAccept"), hydraClient, input.Challenge, hydraLogoutAcceptRequest)
    if err != nil {
      environment.DebugLog(route.LogId, "PostLogout", err.Error(), requestId)
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    environment.DebugLog(route.LogId, "PostLogout", "redirect_to: " + hydraLogoutAcceptResponse.RedirectTo, requestId)
    c.JSON(http.StatusOK, gin.H{
      "redirect_to": hydraLogoutAcceptResponse.RedirectTo,
    })
  }
  return gin.HandlerFunc(fn)
}
