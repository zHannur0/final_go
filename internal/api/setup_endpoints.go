package api

import (
	"final_project/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func setupPublicEndpoints(router *gin.Engine) {
	router.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "welcome to public endpoint"})
	})
}

func setupPrivateEndpoints(router *gin.Engine) {
	router.GET("/private", utils.AuthMiddleware(), func(c *gin.Context) {
		role, _ := c.Get("role")

		if role == "admin" {
			c.JSON(http.StatusOK, gin.H{"message": "welcome to private endpoint (menu admin)"})
		} else if role == "client" {

			c.JSON(http.StatusOK, gin.H{"message": "welcome to private endpoint (user)"})
		}

	})
}
