package api

import (
	"github.com/gin-gonic/gin"
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
