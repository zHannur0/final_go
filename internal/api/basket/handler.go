package basket

import (
	"errors"
	"final_project/initializers"
	"final_project/internal/models"
	"final_project/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"net/http"
)

// DeleteFromBasket godoc
// @Summary Delete user's basket
// @Description Deletes all items in the user's basket and the basket itself.
// @Tags basket
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "message: Basket deleted successfully"
// @Failure 400 {object} map[string]interface{} "error: User ID not found"
// @Failure 404 {object} map[string]interface{} "message: Basket not found"
// @Failure 500 {object} map[string]interface{} "error: Failed to delete basket or basket items"
// @Router /basket [delete]
func DeleteFromBasket(router *gin.Engine) {
	basketRoutes := router.Group("/basket", utils.AuthMiddleware())
	{
		basketRoutes.DELETE("/", func(c *gin.Context) {
			userID, exists := c.Get("ID")
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
				return
			}

			tx := initializers.DB.Begin()

			var basket models.Basket
			if err := tx.Where("user_id = ?", userID.(uint)).First(&basket).Error; err != nil {
				tx.Rollback()
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusOK, gin.H{"message": "Basket not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve basket", "details": err.Error()})
				}
				return
			}

			if err := tx.Where("basket_id = ?", basket.ID).Delete(&models.BasketItem{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete basket items", "details": err.Error()})
				return
			}

			if err := tx.Delete(&basket).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete basket", "details": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Basket deleted successfully"})
		})
	}
}

// AddToBasket godoc
// @Summary Add items to basket
// @Description Adds one or more items to the user's basket.
// @Tags basket
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param items body struct { Items []struct { ItemID uint "json:\"item_id\""; Quantity int "json:\"quantity\"" } "json:\"items\"" } true "Items to add"
// @Success 200 {object} map[string]interface{} "message: Items added to basket successfully, basketId"
// @Failure 400 {object} map[string]interface{} "error: User ID not found or Invalid JSON body"
// @Failure 500 {object} map[string]interface{} "error: Failed to retrieve or create basket or add item to basket"
// @Router /basket [post]
func AddToBasket(router *gin.Engine) {
	basketRoutes := router.Group("/basket", utils.AuthMiddleware())
	{
		basketRoutes.POST("/", func(c *gin.Context) {
			userID, exists := c.Get("ID")
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
				return
			}

			var basketAddRequest struct {
				Items []struct {
					ItemID   uint `json:"item_id"`
					Quantity int  `json:"quantity"`
				} `json:"items"`
			}

			if err := c.BindJSON(&basketAddRequest); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
				return
			}

			tx := initializers.DB.Begin()

			basket := models.Basket{}
			if err := tx.Where("user_id = ?", userID.(uint)).FirstOrCreate(&basket, models.Basket{UserID: userID.(uint)}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve or create basket"})
				return
			}

			for _, item := range basketAddRequest.Items {
				basketItem := models.BasketItem{
					BasketID: basket.ID,
					ItemID:   item.ItemID,
					Quantity: item.Quantity,
				}
				if err := tx.Create(&basketItem).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item to basket"})
					return
				}
			}

			tx.Commit()
			c.JSON(http.StatusOK, gin.H{"message": "Items added to basket successfully", "basketId": basket.ID})
		})
	}
}

// GetAllBasket godoc
// @Summary Retrieve user's basket
// @Description Retrieves all items currently in the user's basket along with total price.
// @Tags basket
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} struct { BasketID uint "json:\"basket_id\""; Items []map[string]interface{} "json:\"items\""; TotalPrice string "json:\"total_price\"" } "Basket contents and total price"
// @Failure 400 {object} map[string]interface{} "error: User ID not found"
// @Failure 404 {object} map[string]interface{} "message: Basket not found"
// @Failure 500 {object} map[string]interface{} "error: Failed to retrieve basket"
// @Router /basket [get]
func GetAllBasket(router *gin.Engine) {
	basketRoutes := router.Group("/basket", utils.AuthMiddleware())
	{
		basketRoutes.GET("/", func(c *gin.Context) {
			userID, exists := c.Get("ID")
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
				return
			}

			var basket models.Basket
			result := initializers.DB.Preload("BasketItems.MenuItem").Where("user_id = ?", userID.(uint)).First(&basket)
			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusOK, gin.H{"basket_id": 0, "items": []interface{}{}, "total_price": "0.00"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve basket", "details": result.Error.Error()})
				return
			}

			var totalPrice decimal.Decimal = decimal.NewFromFloat(0.0)
			items := []map[string]interface{}{}
			for _, item := range basket.BasketItems {
				itemTotalPrice := item.MenuItem.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
				items = append(items, map[string]interface{}{
					"item_id":     item.MenuItem.ID,
					"name":        item.MenuItem.Name,
					"description": item.MenuItem.Description,
					"price":       item.MenuItem.Price.String(),
					"quantity":    item.Quantity,
					"total_price": itemTotalPrice.String(),
				})
				totalPrice = totalPrice.Add(itemTotalPrice)
			}

			if len(items) == 0 {
				c.JSON(http.StatusOK, gin.H{
					"basket_id":   basket.ID,
					"items":       items,
					"total_price": "0.00",
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"basket_id":   basket.ID,
					"items":       items,
					"total_price": totalPrice.String(),
				})
			}
		})
	}
}
