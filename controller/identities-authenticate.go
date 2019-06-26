package controller

import (
  "github.com/gin-gonic/gin"
  "net/http"
  "golang-idp-be/interfaces"
  "golang-idp-be/gateway/hydra"
  _ "os"
  _ "fmt"
)

func PostIdentitiesAuthenticate(c *gin.Context) {

  var input interfaces.PostIdentitiesAuthenticateRequest

  err := c.BindJSON(&input)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    c.Abort()
    return
  }

  hydraLoginResponse, err := hydra.GetLogin(input.Challenge)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    c.Abort()
    return;
  }

  if hydraLoginResponse.Skip {
    hydraLoginAcceptRequest := interfaces.HydraLoginAcceptRequest{
      Subject: hydraLoginResponse.Subject,
      Remember: true,
      RememberFor: 30,
    }

    hydraLoginAcceptResponse := hydra.AcceptLogin(input.Challenge, hydraLoginAcceptRequest)

    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": true,
      "redirect_to": hydraLoginAcceptResponse.RedirectTo,
    })
    c.Abort()
    return
  }


  if input.Id == "user-1" && input.Password == "1234" {
    hydraLoginAcceptRequest := interfaces.HydraLoginAcceptRequest{
      Subject: input.Id,
      Remember: true,
      RememberFor: 30,
    }

    hydraLoginAcceptResponse := hydra.AcceptLogin(input.Challenge, hydraLoginAcceptRequest)

    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": true,
      "redirect_to": hydraLoginAcceptResponse.RedirectTo,
    })
    c.Abort()
    return
  }

  // Deny by default
  c.JSON(http.StatusOK, gin.H{
    "id": input.Id,
    "authenticated": false,
  })
}
