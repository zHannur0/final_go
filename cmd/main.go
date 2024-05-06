package main

import (
	"final_project/initializers"
	"final_project/internal/router"
)

func init() {
	initializers.GetKeysInEnv()
	initializers.ConnectDb()
}

func main() {

	router := router.SetupRouter()
	router.Run(":8080")

}
