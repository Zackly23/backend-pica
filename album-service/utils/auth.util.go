package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AuthTokenJWT(ctx *fiber.Ctx) (*jwt.Token, error) {
	// Ambil token dari header Authorization
	accessToken := ctx.Get("Authorization")
	if accessToken == "" {
		return nil, fmt.Errorf("authorization token tidak ditemukan")
	}

	// Jika pakai "Bearer <token>", hapus prefix
	accessToken = strings.TrimPrefix(accessToken, "Bearer ")


	// Parse dan verifikasi token
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode signing tidak valid: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("token tidak valid atau expired")
	}

	return token, nil
}

func GetUserID(ctx *fiber.Ctx) (uuid.UUID, error) {
	// Validasi dan ambil token
	token, err := AuthTokenJWT(ctx)
	if err != nil {
		return uuid.UUID{},  ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	// Ambil user_id dari token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.UUID{}, ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Gagal mem-parsing klaim token",
		})
	}

	// Gunakan type assertion yang aman
// Ambil user_id dan parse ke uuid.UUID
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.UUID{}, ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User ID tidak ditemukan di token",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.UUID{}, ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Format UUID tidak valid",
		})
	}
	// userID := uint(userIDFloat)

	return userID, nil
}