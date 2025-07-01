package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func CheckHealth(ctx *fiber.Ctx) error {
	fmt.Println("Health check hit!") // Debug log
		if (ctx.Method() != "GET") {
			return fiber.NewError(504, "Method not allowed")
		}
	
		return ctx.JSON(fiber.Map{
			"status": "OK",
	})
}
