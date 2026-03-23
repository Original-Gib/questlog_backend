package main

import (
	"log"
	"os"

	"github.com/Original_Gib/questlog/clients"
	"github.com/Original_Gib/questlog/handlers"
	"github.com/Original_Gib/questlog/services"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env in development; silently ignored if file doesn't exist (prod uses real env vars).
	_ = godotenv.Load()

	clientID := os.Getenv("TWITCH_CLIENT_ID")
	clientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("TWITCH_CLIENT_ID and TWITCH_CLIENT_SECRET must be set")
	}

	igdbClient := clients.NewIGDBClient(clientID, clientSecret)
	igdbService := services.NewIGDBService(igdbClient)
	gamesHandler := handlers.NewGamesHandler(igdbService)

	r := gin.Default()

	r.GET("/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello, Gamer!"})
	})

	v1 := r.Group("/api/v1")
	gamesHandler.RegisterRoutes(v1)

	r.Run(":8080")
}
