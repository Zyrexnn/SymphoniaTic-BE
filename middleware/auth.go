package middleware

import (
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

// RequireUserAuth middleware ensures the request has a valid Bearer JWT token.
func RequireUserAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Akses ditolak. Header Authorization tidak ditemukan.",
		})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Format token tidak valid. Gunakan format 'Bearer <token>'.",
		})
	}

	claims, err := utils.ValidateUserToken(parts[1])
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Token tidak valid atau telah kedaluwarsa.",
			Error:   err.Error(),
		})
	}

	c.Locals("user_id", claims.UserID)
	c.Locals("user_email", claims.Email)
	c.Locals("user_role", claims.Role)

	return c.Next()
}
