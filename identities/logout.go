package identities

import (
  "net/http"
  "fmt"

  "github.com/gin-gonic/gin"

  "golang-idp-be/config"
  "golang-idp-be/gateway/idpbe"
  "golang-idp-be/gateway/hydra"
)

type LogoutRequest struct {
  Challenge       string            `json:"challenge" binding:"required"`
}

type LogoutResponse struct {
}

func PostLogout(env *idpbe.IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.PostLogout]", c.MustGet("RequestId")))
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
    hydraLogoutAcceptResponse, err := hydra.AcceptLogout(config.Hydra.LogoutRequestAcceptUrl, hydraClient, input.Challenge, hydraLogoutAcceptRequest)
    if err != nil {
      fmt.Println("identities.PostLogout:" + err.Error())
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    fmt.Println("identities.PostLogout, redirect_to:" + hydraLogoutAcceptResponse.RedirectTo)
    c.JSON(http.StatusOK, gin.H{
      "redirect_to": hydraLogoutAcceptResponse.RedirectTo,
    })
  }
  return gin.HandlerFunc(fn)
}
