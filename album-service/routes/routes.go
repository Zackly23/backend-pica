package routes

import (
	"fmt"
	"os"
	"strings"

	notif "github.com/Zackly23/queue-app/proto/notificationpb"

	"github.com/Zackly23/queue-app/handlers"
	"github.com/Zackly23/queue-app/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

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
		return handlers.SignUp(c, db)
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
		return handlers.VerifyTOTP(c, db)
	})

	// Protected routes (dengan JWT middleware)
	authRoutes := v1.Group("/", JWTMiddleware(db))

	authRoutes.Post("/logout", func(c *fiber.Ctx) error {
		return handlers.Logout(c, db)
	})

	userRoutes := authRoutes.Group("/users")


	userRoutes.Delete("/deactivate", func(c *fiber.Ctx) error {
		return handlers.DeactivateAccount(c, db)
	})

	userRoutes.Delete("/delete", func(c *fiber.Ctx) error {
		return handlers.DeleteAccount(c, db)
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

	albumRoutes := authRoutes.Group("/albums")
	
	albumRoutes.Post("/", func(c *fiber.Ctx) error {
		return handlers.StoreAlbums(c, db)
	})
	
	//ini /albums dulunya
	albumRoutes.Get("/", func(c *fiber.Ctx) error {
		return handlers.GetAllAlbums(c, db)
	})

	albumRoutes.Post("/media", func(c *fiber.Ctx) error {
		return handlers.UploadMediaAlbum(c, db)
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

	albumRoutes.Put("/:albumId/target", func(c *fiber.Ctx) error {
		return handlers.UpdateTargetEmail(c, db)
	})

	albumRoutes.Get("/:albumId", func(c *fiber.Ctx) error {
		return handlers.GetAlbum(c, db)
	})

	albumRoutes.Put("/:albumID", func(c *fiber.Ctx) error {
		return handlers.UpdateAlbum(c, db)
	})
	// Tambahkan route lain yang butuh proteksi di sini
}

