package identities

import (
  "net/http"
  "fmt"

  "github.com/gin-gonic/gin"

  "golang-idp-be/gateway/hydra"
)

type LogoutRequest struct {
  Challenge       string            `json:"challenge" binding:"required"`
}

type LogoutResponse struct {
}

func PostLogout(env *IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.PostLogout]", c.MustGet("RequestId")))
    var input LogoutRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    hydraLogoutAcceptRequest := hydra.HydraLogoutAcceptRequest{
    }
    hydraLogoutAcceptResponse, err := hydra.AcceptLogout(input.Challenge, hydraLogoutAcceptRequest)
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
