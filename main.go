package main

import (
	"context"
	"os"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"backend/database"
	"github.com/gin-contrib/cors"
	routes "backend/routes"
)

var client *mongo.Client


func mongoFun(c *gin.Context){
	c.String(200,"hello")
}



func main() {
	router := gin.Default()
	client = database.DBinstance()
	defer client.Disconnect(context.Background())
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
	config.AllowHeaders = []string{"Authorization", "Content-Type"}
	router.Use(cors.New(config))

	routes.ClientRoutes(router) 

	routes.AuthRoutes(router)
	routes.ClientAuthRoutes(router)
	routes.UserRoutes(router)

	router.Use(gin.Logger())


	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
