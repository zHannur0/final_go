package menu

import (
	"final_project/initializers"
	"final_project/internal/models"
	"final_project/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetAllMenu godoc
// @Summary Get all menu items
// @Description Retrieves all available menu items from the database.
// @Tags menu
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} struct { MenuItems []models.Menu } "A list of all menu items"
// @Failure 500 {object} map[string]interface{} "error: Failed to retrieve menu items"
// @Router /menu [get]
func GetAllMenu(router *gin.Engine) {
	menuRoutes := router.Group("/menu", utils.AuthMiddleware())
	{
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

// AddMenu godoc
// @Summary Add a new menu item
// @Description Adds a new menu item to the database, accessible only by admin users.
// @Tags menu
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param menuItem body models.Menu true "Menu Item to be added"
// @Success 201 {object} map[string]interface{} "message: Menu item added successfully, menuItemId"
// @Failure 400 {object} map[string]interface{} "error: Invalid request"
// @Failure 403 {object} map[string]interface{} "error: Insufficient permissions"
// @Failure 500 {object} map[string]interface{} "error: Failed to add menu item"
// @Router /menu [post]
func AddMenu(router *gin.Engine) {
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
	}
}

// UpdateMenu godoc
// @Summary Update a menu item
// @Description Updates details of a specific menu item, accessible only by admin users.
// @Tags menu
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param itemId path string true "ID of the Menu Item to update"
// @Param updates body map[string]interface{} true "JSON object containing the updates"
// @Success 200 {object} map[string]interface{} "message: Menu item updated successfully"
// @Failure 400 {object} map[string]interface{} "error: Invalid request, details"
// @Failure 403 {object} map[string]interface{} "error: Insufficient permissions"
// @Failure 404 {object} map[string]interface{} "error: Menu item not found"
// @Failure 500 {object} map[string]interface{} "error: Failed to update menu item, details"
// @Router /menu/{itemId} [patch]
func UpdateMenu(router *gin.Engine) {
	menuRoutes := router.Group("/menu", utils.AuthMiddleware())
	{
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
	}
}

// DeleteMenu godoc
// @Summary Delete a menu item
// @Description Deletes a specific menu item from the database, accessible only by admin users.
// @Tags menu
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param itemId path string true "ID of the Menu Item to delete"
// @Success 200 {object} map[string]interface{} "message: Menu item deleted successfully"
// @Failure 403 {object} map[string]interface{} "error: Insufficient permissions"
// @Failure 500 {object} map[string]interface{} "error: Failed to delete menu item"
// @Router /menu/{itemId} [delete]
func DeleteMenu(router *gin.Engine) {
	menuRoutes := router.Group("/menu", utils.AuthMiddleware())
	{
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
	}
}
