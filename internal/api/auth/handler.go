package auth

import (
	"final_project/initializers"
	"final_project/internal/models"
	"final_project/internal/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
)

func Login(router *gin.Engine) {
	router.POST("/login", func(c *gin.Context) {
		var loginUser models.User
		if err := c.BindJSON(&loginUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		var existingUser models.User
		result := initializers.DB.Select("ID", "username", "password", "role").Where("username = ?", loginUser.Username).First(&existingUser)
		if result.Error != nil || !utils.CheckPassword(existingUser.Password, loginUser.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		token, err := utils.GenerateToken(existingUser.Username, string(existingUser.Role), int(existingUser.ID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		fmt.Println("Generated token:", token)
		c.JSON(http.StatusOK, gin.H{"token": token})
	})
}

func SignUp(router *gin.Engine) {

	router.POST("/signup", func(c *gin.Context) {
		var newUser models.User
		if err := c.BindJSON(&newUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		validate := validator.New()

		if err := validate.Struct(newUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
			return
		}
		if err := utils.SignupUser(initializers.DB, newUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign up user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "User signed up successfully"})
	})
}
