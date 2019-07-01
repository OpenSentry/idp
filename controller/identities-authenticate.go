package controller

import (
  _ "os"
  "fmt"
  "net/http"

  "github.com/gin-gonic/gin"

  "golang-idp-be/interfaces"
  "golang-idp-be/gateway/hydra"
)

func PostIdentitiesAuthenticate(c *gin.Context) {
  fmt.Println(fmt.Sprintf("[request-id:%s][event:PostIdentitiesAuthenticate]", c.MustGet("RequestId")))

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

    fmt.Println("IdpBe.PostIdentitiesAuthenticate, id:"+input.Id+" authenticated:true redirect_to:"+hydraLoginAcceptResponse.RedirectTo)
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

    fmt.Println("IdpBe.PostIdentitiesAuthenticate, id:"+input.Id+" authenticated:true redirect_to:"+hydraLoginAcceptResponse.RedirectTo)
    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": true,
      "redirect_to": hydraLoginAcceptResponse.RedirectTo,
    })
    c.Abort()
    return
  }

  // Deny by default
  fmt.Println("IdpBe.PostIdentitiesAuthenticate, id:"+input.Id+" authenticated:false redirect_to:")
  c.JSON(http.StatusOK, gin.H{
    "id": input.Id,
    "authenticated": false,
  })
  c.Abort()
}
