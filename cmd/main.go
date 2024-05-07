package main

import (
	_ "final_project/docs"
	"final_project/initializers"
	"final_project/internal/api"
)

func init() {
	initializers.GetKeys()
	initializers.DBConnector()
}

// @title Canteen SDU
// @version 1.0
// @description API Server for Canteen

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apiKey ApiKeyAuth
// @in header
// @name Authorization

func main() {

	router := api.SetupRouter()
	router.Run(":8080")

}
