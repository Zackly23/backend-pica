package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	"github.com/joho/godotenv"

	"github.com/Zackly23/queue-app/config"
	notif "github.com/Zackly23/queue-app/proto/notificationpb"

	"github.com/Zackly23/queue-app/routes"
)

func main() {
	// Load .env
	var databaseInstance config.Database
	var db *gorm.DB


	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	// Connect DB + Redis
	db, err = databaseInstance.ConnectDatabase()
	config.ConnectRedis()

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connected successfully")

	conn, err := grpc.Dial(":50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }

    defer conn.Close()

    client := notif.NewNotificationServiceClient(conn)

	// Init Fiber
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowCredentials: true,
	}))


	// Register routes
	routes.SetupRoutes(app, db, client)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Println("Server running on port:", port)
	log.Fatal(app.Listen(":" + port))

}


