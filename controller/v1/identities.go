package controller

import (
  "github.com/gin-gonic/gin"
  _ "golang-idp-be/config"
  "net/http"
  _ "os"
  _ "fmt"
)

func GetIdentities(c *gin.Context) {

  id, _ := c.GetQuery("id")

  if id == "user-1" {
    c.JSON(http.StatusOK, gin.H{
      "id": id,
      "name": "Test bruger",
      "email": "test@test.dk",
    })
    return
  }

  // Deny by default
  c.JSON(http.StatusNotFound, gin.H{
    "error": "Not found",
  })
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
