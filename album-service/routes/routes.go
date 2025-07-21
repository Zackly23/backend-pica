package routes

import (
	"fmt"
	"os"
	"strings"

	notif "github.com/Zackly23/queue-app/proto/notificationpb"
	"github.com/Zackly23/queue-app/utils"

	"github.com/Zackly23/queue-app/handlers"
	"github.com/Zackly23/queue-app/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func DynamicStorageCapacityMiddleware(db *gorm.DB) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		userID, err := utils.GetUserID(ctx)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Ambil user beserta data subscription
		var user models.User
		if err := db.Preload("Subscription").First(&user, "id = ?", userID).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal mengambil data subscription user",
			})
		}

		// Ambil semua album beserta isinya
		var albums []models.Album
		if err := db.Preload("AlbumImages").Preload("AlbumVideos").Where("user_id = ?", user.ID).Find(&albums).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal mengambil data album user",
			})	
		}

		// Hitung total penggunaan penyimpanan
		// Hitung total penggunaan penyimpanan
		var storageUsed float64 = 0.0
		for _, album := range albums {
			for _, image := range album.AlbumImages {
				storageUsed += float64(image.Size) // dalam MB
			}
			for _, video := range album.AlbumVideos {
				storageUsed += float64(video.Size) // dalam MB
			}
		}

		storageCapacity := float64(user.Subscription.StorageCapacity) * 1024 // dari GB ke MB

		fmt.Println("Storage Capacity:", storageCapacity, "MB")
		// Cek apakah storage sudah penuh
		if storageCapacity > 0 {
			percentageUsed := (storageUsed / storageCapacity) * 100.0
			fmt.Printf("Storage Used: %.2f MB, Percentage Used: %.2f%%\n", storageUsed, percentageUsed)
			if percentageUsed >= 100 {
				return ctx.Status(fiber.StatusUnavailableForLegalReasons).JSON(fiber.Map{
					"error": fmt.Sprintf("Kapasitas album sudah penuh. Maksimum %.2f GB", user.Subscription.StorageCapacity),
				})
			}
		}

		return ctx.Next()
	}
}

func DynamicBodyLimitMiddleware(db *gorm.DB) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		userID, err := utils.GetUserID(ctx)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Get user's subscription info
		var user models.User
		if err := db.Preload("Subscription").First(&user, "id = ?", userID).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch user subscription",
			})
		}

		// Read Content-Length from header
		contentLength := ctx.Request().Header.ContentLength()
		maxAllowedSize := int(user.Subscription.MaximumMediaSize * 1024 * 1024 * 100) // Convert MB to Bytes

		if contentLength > maxAllowedSize {
			return ctx.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error": fmt.Sprintf("File terlalu besar. Maksimum %v MB", user.Subscription.MaximumMediaSize),
			})
		}

		// Lanjut ke handler
		return ctx.Next()
	}
}


func JWTMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		secret := os.Getenv("JWT_SECRET_KEY")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token tidak valid",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token tidak valid",
			})
		}

		// Cek apakah token sudah direvoke
		var tokenRecord models.PersonalAccessToken
		if err := db.Where("access_token = ? AND revoked = false", tokenStr).First(&tokenRecord).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token sudah tidak berlaku atau di-revoke",
			})
		}

		// Simpan data user ke context
		c.Locals("user_id", claims["user_id"])
		c.Locals("email", claims["email"])
		return c.Next()
	}
}


func SetupRoutes(app *fiber.App, db *gorm.DB, client notif.NotificationServiceClient) {
	fmt.Println("Setting up routes...")

	api := app.Group("/api")
	v1 := api.Group("/v1")

	v1.Post("/temp/image", func(c *fiber.Ctx) error {
		return handlers.UploadTemporary(c, db)
	})
	
	// Public routes (tanpa JWT)
	auth := v1.Group("/auth")
	
	auth.Get("/health", handlers.CheckHealth)

	auth.Post("/login", func(c *fiber.Ctx) error {
		return handlers.Login(c, db)
	})
	auth.Post("/signup", func(c *fiber.Ctx) error {
		return handlers.SignUp(c, db, client)
	})
	auth.Get("/refresh", func(c *fiber.Ctx) error {
		return handlers.Refresh(c, db)
	})

	auth.Post("/reset-password", func(c *fiber.Ctx) error {
		return handlers.ResetPassword(c, db, client)
	})


	auth.Put("/change-password", func(c *fiber.Ctx) error {
		return handlers.ChangePassword(c,db, client)
	})

	auth.Post("/generate-totp", func(c *fiber.Ctx) error {
		return handlers.GenerateTOTP(c,db)
	})

	auth.Post("/verify-totp", func(c *fiber.Ctx) error {
		return handlers.VerifyTOTP(c, db, client)
	})

	auth.Post("/verify-tfa", func(c *fiber.Ctx) error {
		return handlers.VerifyTFA(c, db, client)
	})

	// Protected routes (dengan JWT middleware)
	authRoutes := v1.Group("/", JWTMiddleware(db))

	authRoutes.Post("/logout", func(c *fiber.Ctx) error {
		return handlers.Logout(c, db)
	})

	userRoutes := authRoutes.Group("/users")

	userRoutes.Post("/follow", func(c *fiber.Ctx) error {
		return handlers.FollowUser(c, db)
	})

	userRoutes.Get("/subscription", func(c *fiber.Ctx) error {
		return handlers.GetSubscriptionHistory(c, db)
	})

	userRoutes.Delete("/deactivate", func(c *fiber.Ctx) error {
		return handlers.DeactivateAccount(c, db, client)
	})

	userRoutes.Delete("/delete", func(c *fiber.Ctx) error {
		return handlers.DeleteAccount(c, db, client)
	})

	userRoutes.Get("/:userId", func(c *fiber.Ctx) error {
		return handlers.GetUserData(c, db)
	})

	userRoutes.Put("/:userId", func(c *fiber.Ctx) error {
		return handlers.UpdateUserData(c, db)
	})

	userRoutes.Get("/:userId/configuration", func(c *fiber.Ctx) error {
		return handlers.GetUserConfiguration(c, db)
	})


	userRoutes.Put("/:userId/profile/picture", func(c *fiber.Ctx) error {
		return handlers.UpdateProfilePicture(c, db)
	})

	albumRoutes := authRoutes.Group("/albums", DynamicBodyLimitMiddleware(db))
	
	albumRoutes.Post("/", DynamicStorageCapacityMiddleware(db), func(c *fiber.Ctx) error {
		return handlers.StoreAlbums(c, db, client)
	})
	
	//ini /albums dulunya
	albumRoutes.Get("/", func(c *fiber.Ctx) error {
		return handlers.GetAllAlbums(c, db)
	})

	albumRoutes.Post("/media", DynamicStorageCapacityMiddleware(db), func(c *fiber.Ctx) error {
		return handlers.UploadMediaAlbum(c, db)
	})

	albumRoutes.Get("/media/follower", func(c *fiber.Ctx) error {
		return handlers.GetAlbumFollower(c, db)
	})

	albumRoutes.Get("/comments", func(c *fiber.Ctx) error {
		return handlers.GetAlbumComments(c, db)
	})

	albumRoutes.Post("/comments", func(c *fiber.Ctx) error {
		return handlers.PostAlbumComment(c, db)
	})

	albumRoutes.Post("/likes", func(c *fiber.Ctx) error {
		return handlers.ClickLikeAlbum(c, db)
	})

	albumRoutes.Post("/media/likes", func(c *fiber.Ctx) error {
		return handlers.ClickLikeMedia(c, db)
	})
	
	albumRoutes.Get("/images/latest", func(c *fiber.Ctx) error {
		return handlers.GetLatestImage(c, db)
	})

	albumRoutes.Put("/:albumId/target-email", func(c *fiber.Ctx) error {
		return handlers.UpdateTargetEmail(c, db, client)
	})

	albumRoutes.Get("/:albumId", func(c *fiber.Ctx) error {
		return handlers.GetAlbum(c, db)
	})

	albumRoutes.Put("/:albumID", DynamicStorageCapacityMiddleware(db), func(c *fiber.Ctx) error {
		return handlers.UpdateAlbum(c, db)
	})

	albumRoutes.Delete("/:albumID", func(c *fiber.Ctx) error {
		return handlers.DeleteAlbum(c, db)
	})
	// Tambahkan route lain yang butuh proteksi di sini
}

