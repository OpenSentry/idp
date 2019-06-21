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
  var input interfaces.PostIdentitiesLogoutRequest

  err := c.BindJSON(&input)

  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    c.Abort()
    return
  }

  hydraLogoutResponse, err := hydra.GetLogout(input.Challenge)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    c.Abort()
    return
  }

  fmt.Println(hydraLogoutResponse)

  hydraLogoutAcceptRequest := interfaces.HydraLogoutAcceptRequest{
  }
  hydraLogoutAcceptResponse, err := hydra.AcceptLogout(input.Challenge, hydraLogoutAcceptRequest)

  c.JSON(http.StatusOK, gin.H{
    "redirect_to": hydraLogoutAcceptResponse.RedirectTo,
  })
}
