package main

import (
	"final_project/initializers"
	"final_project/internal/api"
)

func init() {
	initializers.GetKeysInEnv()
	initializers.ConnectDb()
}

func main() {

	router := api.SetupRouter()
	router.Run(":8080")

}
