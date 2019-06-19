package controller

import (
  "github.com/gin-gonic/gin"
  _ "golang-idp-be/config"
  "net/http"
  _ "os"
  _ "fmt"
)

func GetIdentities(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "message": "pong",
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
