package api

import (
	"final_project/initializers"
	"final_project/internal/models"
	"final_project/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func setupMenuEndpoints(router *gin.Engine) {
	menuRoutes := router.Group("/menu", utils.AuthMiddleware())
	{
		menuRoutes.POST("/", func(c *gin.Context) {
			role, _ := c.Get("role")
			if role == "admin" {
				var menuItem models.Menu
				if err := c.BindJSON(&menuItem); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
					return
				}
				if err := initializers.DB.Create(&menuItem).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add menu item"})
					return
				}
				c.JSON(http.StatusCreated, gin.H{"message": "Menu item added successfully", "menuItemId": menuItem.ID})
			} else if role == "client" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				return
			}
		})

		menuRoutes.PATCH("/:itemId", func(c *gin.Context) {
			role, _ := c.Get("role")
			if role != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				return
			}

			itemId := c.Param("itemId")
			var updates map[string]interface{}
			if err := c.BindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
				return
			}

			result := initializers.DB.Model(&models.Menu{}).Where("id = ?", itemId).Updates(updates)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update menu item", "details": result.Error.Error()})
				return
			}

			if result.RowsAffected == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Menu item updated successfully"})
		})

		menuRoutes.DELETE("/:itemId", func(c *gin.Context) {
			role, _ := c.Get("role")
			if role != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				return
			}
			itemId := c.Param("itemId")
			if err := initializers.DB.Delete(&models.Menu{}, itemId).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete menu item"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Menu item deleted successfully"})
		})

		menuRoutes.GET("/", func(c *gin.Context) {
			var menuItems []models.Menu
			if err := initializers.DB.Find(&menuItems).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve menu items"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"menuItems": menuItems})
		})
	}
}
