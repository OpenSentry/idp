package controller

import (
  _ "os"
  "fmt"
  "net/http"

  "github.com/gin-gonic/gin"

  _ "golang-idp-be/config"
)

func GetIdentities(c *gin.Context) {
  fmt.Println(fmt.Sprintf("[request-id:%s][event:GetIdentities]", c.MustGet("RequestId")))

  id, _ := c.GetQuery("id")

  if id == "user-1" {
    c.JSON(http.StatusOK, gin.H{
      "id": id,
      "name": "Test bruger",
      "email": "test@test.dk",
    })
    c.Abort()
    return
  }

  // Deny by default
  c.JSON(http.StatusNotFound, gin.H{
    "error": "Not found",
  })
  c.Abort()
}

func PostIdentities(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
  })
}

func PutIdentities(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
  })
}

func PostIdentitiesRevoke(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
  })
}

func PostIdentitiesRecover(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
  })
}
