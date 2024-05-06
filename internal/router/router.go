package router

import (
	"errors"
	"final_project/initializers"
	"final_project/internal/models"
	"final_project/internal/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"net/http"
	"time"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()
	setupPublicEndpoints(router)
	setupPrivateEndpoints(router)
	setupAuthEndpoints(router)
	setupBasketRouters(router)

	setupMenuEndpoints(router)
	setupOrderEndpoints(router)
	setupOrderRoutes(router)
	SetupOrderDeleteRouter(router)

	//only admin routers
	SetupOrderUpdateRouter(router)

	return router
}

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

func setupAuthEndpoints(router *gin.Engine) {
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

type BasketItemUpdateRequest struct {
	ItemID   uint `json:"item_id"`
	Quantity int  `json:"quantity"`
}

func setupBasketRouters(router *gin.Engine) {
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

type OrderRequest struct {
	OrderItems []OrderItem `json:"order_items"`
}

type OrderItem struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}

func setupOrderEndpoints(router *gin.Engine) {
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

func setupOrderRoutes(router *gin.Engine) {
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

type UpdateOrderData struct {
	Status string `json:"status" binding:"required"`
}

func SetupOrderUpdateRouter(router *gin.Engine) {
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
func SetupOrderDeleteRouter(router *gin.Engine) {
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
