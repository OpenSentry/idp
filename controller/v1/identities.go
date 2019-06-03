package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"idp/gateway/db"
) 

func GetIdentities(c *gin.Context) {

	result, err := db.HelloWorld("bolt://neo4j", "neo4j", "test")

	c.JSON(http.StatusOK, gin.H{
		"message": result,
		"error": err,
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

func GetIdentitiesAuthenticate(c *gin.Context) {
	id := c.Query("id")
	password := c.Query("password")
	message := "id: " + id + ", password: " + password
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
}

func GetIdentitiesRevoke(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func GetIdentitiesRecover(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}