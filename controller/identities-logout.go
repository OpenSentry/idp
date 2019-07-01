package controller

import (
  "github.com/gin-gonic/gin"
  "net/http"
  "golang-idp-be/interfaces"
  "golang-idp-be/gateway/hydra"
  _ "os"
  "fmt"
)

func PostIdentitiesLogout(c *gin.Context) {
  fmt.Println(fmt.Sprintf("[request-id:%s][event:PostIdentitiesLogout]", c.MustGet("RequestId")))
  var input interfaces.PostIdentitiesLogoutRequest

  err := c.BindJSON(&input)

  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    c.Abort()
    return
  }

  hydraLogoutAcceptRequest := interfaces.HydraLogoutAcceptRequest{
  }
  hydraLogoutAcceptResponse, err := hydra.AcceptLogout(input.Challenge, hydraLogoutAcceptRequest)

  fmt.Println("IdpBe.PostIdentitiesLogout, redirect_to:" + hydraLogoutAcceptResponse.RedirectTo)
  c.JSON(http.StatusOK, gin.H{
    "redirect_to": hydraLogoutAcceptResponse.RedirectTo,
  })
  c.Abort()
}
