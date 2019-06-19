package controller

import (
  "github.com/gin-gonic/gin"
  "net/http"
  "golang-idp-be/interfaces"
  "golang-idp-be/gateway/hydra"
  _ "os"
  "fmt"
)

func PostIdentitiesAuthenticate(c *gin.Context) {

  var input interfaces.PostIdentitiesAuthenticateRequest

  err := c.BindJSON(&input)

  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  hydraLoginResponse, err := hydra.GetLogin(input.Challenge)

  if err != nil {
    fmt.Println(err)
  }

  if hydraLoginResponse.Skip {
    hydraLoginAcceptRequest := interfaces.HydraLoginAcceptRequest{
      Subject: hydraLoginResponse.Subject,
    }

    hydraLoginAcceptResponse := hydra.AcceptLogin(input.Challenge, hydraLoginAcceptRequest)

    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": true,
      "redirect_to": hydraLoginAcceptResponse.RedirectTo,
    })

    return
  }


  if input.Id == "user-1" && input.Password == "1234" {
    hydraLoginAcceptRequest := interfaces.HydraLoginAcceptRequest{
      Subject: input.Id,
    }

    hydraLoginAcceptResponse := hydra.AcceptLogin(input.Challenge, hydraLoginAcceptRequest)

    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": true,
      "redirect_to": hydraLoginAcceptResponse.RedirectTo,
    })

    return
  }

  c.JSON(http.StatusOK, gin.H{
    "id": input.Id,
    "authenticated": false,
  })
}
