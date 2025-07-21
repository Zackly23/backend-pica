package main

import (
	"fmt"
	"log"
	"os"
	"time"

	// "time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	"github.com/Zackly23/queue-app/config"
	"github.com/Zackly23/queue-app/jobs"
	notif "github.com/Zackly23/queue-app/proto/notificationpb"

	// "github.com/Zackly23/queue-app/wasabi"

	"github.com/Zackly23/queue-app/routes"
)

func main() {
	// Load .env
	var databaseInstance config.Database
	var db *gorm.DB
	var bucket config.AWSS3Bucket

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}


	//setup s3

	bucket = config.AWSS3Bucket{
		BucketName: "your-bucket-name",
		Region:     "ap-southeast-1", // contoh region
	}

	bucket.SetupBucket()
	bucket.Test()
	
	// Connect DB + Redis
	db, err = databaseInstance.ConnectDatabase()
	config.ConnectRedis()

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connected successfully")

	// setup cron job
	cronJob := cron.New(cron.WithLocation(time.FixedZone("Asia/Jakarta", 7*60*60)))

	cronJob.AddFunc("0 2 * * *", func() {
		log.Println("Menjalankan cron: CleanUpUnusedFiles")
		jobs.CleanUpUnusedFiles(db)
	})

	cronJob.AddFunc("0 0 * * *", func() {
		log.Println("Menjalankan cron: UpdateSubscriptionType User")
		jobs.UpdateSubscriptionType(db)
	})

	// cronJob.AddFunc("@every 1m", func() {
	// 	log.Println("Menjalankan cron setiap 1 menit (testing)")
	// })

	cronJob.Start()


	conn, err := grpc.NewClient(":50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
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

		// Cek semua route saat app jalan
	for _, route := range app.GetRoutes() {
		fmt.Printf("Method: %-6s Path: %s\n", route.Method, route.Path)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Println("Server running on port:", port)
	log.Fatal(app.Listen(":" + port))

}


