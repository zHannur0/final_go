package utils

import (
	"final_project/internal/models"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"gorm.io/gorm"
)

var jwtKey = []byte(os.Getenv("my_secret"))

func GenerateToken(username string, role string, ID int) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"role":     role,
		"ID":       ID,
		"exp":      time.Now().Add(time.Hour * 1).Unix(),
	})

	return token.SignedString(jwtKey)
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
	  tokenString := c.GetHeader("Authorization")
	  if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	  }
  
	  tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
	  token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		  return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	  })
  
	  if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	  }
  
	  claims, ok := token.Claims.(jwt.MapClaims)
	  if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		c.Abort()
		return
	  }
  
	  userID, ok := claims["ID"].(float64)  // JWT numeric values are float64
	  if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		c.Abort()
		return
	  }
  
	  role, ok := claims["role"].(string)
	  if !ok || (role != "client" && role != "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
		return
	  }
  
	  c.Set("role", role)
	  c.Set("ID", uint(userID))  
	  c.Next()
	}
  }
  
func SignupUser(db *gorm.DB, newUser models.User) error {

	var existingUser models.User
	result := db.Where("username = ?", newUser.Username).First(&existingUser)
	if result.Error == nil {
		return fmt.Errorf("Username already exists")
	}
	hashedPassword, err := HashPassword(newUser.Password)
	if err != nil {
		return fmt.Errorf("Failed to hash password")
	}
	newUser.Password = hashedPassword
	if err := db.Create(&newUser).Error; err != nil {
		return fmt.Errorf("Failed to create user")
	}

	return nil
}
