package order

import (
	"final_project/initializers"
	"final_project/internal/models"
	"final_project/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"net/http"
	"time"
)

func AddOrder(router *gin.Engine) {
	orders := router.Group("/orders", utils.AuthMiddleware())
	{
		orders.POST("/", func(c *gin.Context) {
			userID, _ := c.Get("ID")
			var orderReq OrderRequest
			if err := c.BindJSON(&orderReq); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
				return
			}

			newOrder := models.Order{
				UserID:       userID.(uint),
				OrderDetails: []models.OrderDetail{},
				CreatedAt:    time.Now(),
				OrderStatus:  models.Preparing,
			}

			var totalPrice decimal.Decimal
			tx := initializers.DB.Begin()

			for _, item := range orderReq.OrderItems {
				menuItem := models.Menu{}
				result := tx.First(&menuItem, item.ProductID)
				if result.Error != nil {
					tx.Rollback()
					c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found", "productID": item.ProductID})
					return
				}

				if menuItem.Quantity < item.Quantity {
					tx.Rollback()
					c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough stock", "productID": item.ProductID})
					return
				}

				menuItem.Quantity -= item.Quantity
				if err := tx.Save(&menuItem).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update menu item stock", "productID": item.ProductID})
					return
				}

				itemTotalCost := menuItem.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))

				orderDetail := models.OrderDetail{
					ItemID:    item.ProductID,
					Quantity:  item.Quantity,
					TotalCost: itemTotalCost,
				}
				newOrder.OrderDetails = append(newOrder.OrderDetails, orderDetail)

				totalPrice = totalPrice.Add(itemTotalCost)
			}

			newOrder.TotalPrice = totalPrice

			if err := tx.Create(&newOrder).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
				return
			}

			tx.Commit()
			c.JSON(http.StatusCreated, newOrder)
		})
	}
}

func GetOrder(router *gin.Engine) {
	orders := router.Group("/orders", utils.AuthMiddleware())
	{
		orders.GET("/", func(c *gin.Context) {
			userID, exists := c.Get("ID")
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
				return
			}

			var userOrders []models.Order
			if err := initializers.DB.Preload("OrderDetails.MenuItem").Where("user_id = ?", userID.(uint)).Find(&userOrders).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders", "details": err.Error()})
				return
			}

			response := make([]map[string]interface{}, 0)
			for _, order := range userOrders {
				orderItems := make([]map[string]interface{}, 0)
				for _, detail := range order.OrderDetails {
					orderItems = append(orderItems, map[string]interface{}{
						"id": detail.ID,
						"item": map[string]interface{}{
							"ID":          detail.MenuItem.ID,
							"name":        detail.MenuItem.Name,
							"description": detail.MenuItem.Description,
							"price":       detail.MenuItem.Price.String(),
						},
						"quantity":    detail.Quantity,
						"total_price": detail.TotalCost.String(),
					})
				}

				response = append(response, map[string]interface{}{

					"order_id":     order.ID,
					"order_items":  orderItems,
					"order_status": order.OrderStatus,
					"order_cost":   order.TotalPrice.String(),
					"updated_at":   order.UpdatedAt.Format(time.RFC3339Nano),
					"created_at":   order.CreatedAt.Format(time.RFC3339Nano),
				})
			}

			c.JSON(http.StatusOK, response)
		})
	}
}

func UpdateOrder(router *gin.Engine) {
	orders := router.Group("/orders", utils.AuthMiddleware())
	{
		orders.PATCH("/:OrderId", func(c *gin.Context) {
			var updateData UpdateOrderData
			if err := c.BindJSON(&updateData); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
				return
			}

			role, _ := c.Get("role")
			if role != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can update order status"})
				return
			}

			orderID := c.Param("OrderId")
			orderStatus := models.Status(updateData.Status)

			switch orderStatus {
			case models.Canceled, models.Preparing, models.Ready, models.Completed:
				order := &models.Order{}
				result := initializers.DB.First(order, orderID)
				if result.Error != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
					return
				}

				order.OrderStatus = orderStatus

				// Сохраняем изменения и выполняем проверку перед сохранением
				if err := initializers.DB.Save(order).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status", "details": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{"message": "Order status updated successfully"})
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order status"})
			}
		})
	}
}

func DeleteOrder(router *gin.Engine) {
	orders := router.Group("/orders", utils.AuthMiddleware())
	{
		orders.DELETE("/:OrderID/", func(c *gin.Context) {
			userID, exists := c.Get("ID")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
				return
			}
			orderID := c.Param("OrderID")
			var order models.Order
			result := initializers.DB.Where("id = ? AND user_id = ?", orderID, userID.(uint)).First(&order)
			if result.Error != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Order not found or you don't have permission to delete it"})
				return
			}
			if order.OrderStatus != models.Preparing {
				c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete orders with 'preparing' status"})
				return
			}
			if err := initializers.DB.Delete(&order).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
		})
	}
}

type UpdateOrderData struct {
	Status string `json:"status" binding:"required"`
}

type OrderRequest struct {
	OrderItems []OrderItem `json:"order_items"`
}

type OrderItem struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}
