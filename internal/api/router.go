package api

import (
	_ "final_project/docs"
	"final_project/internal/api/auth"
	"final_project/internal/api/basket"
	"final_project/internal/api/menu"
	"final_project/internal/api/order"
	"final_project/internal/api/status"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	//status
	status.PublicStatus(router)
	status.PrivateStatus(router)

	//auth
	auth.Login(router)
	auth.SignUp(router)

	//basket
	basket.GetAllBasket(router)
	basket.DeleteFromBasket(router)
	basket.AddToBasket(router)

	// menu
	menu.GetAllMenu(router)
	menu.AddMenu(router)
	menu.UpdateMenu(router)
	menu.DeleteMenu(router)

	order.AddOrder(router)
	order.GetOrder(router)
	order.DeleteOrder(router)
	order.UpdateOrder(router)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}
