package middleware

import (
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

// RequireUserAuth middleware ensures the request has a valid Bearer JWT token.
func RequireUserAuth(ctx *fiber.Ctx) error {
	rawAuthHeader := ctx.Get("Authorization")
	if rawAuthHeader == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Akses ditolak. Header Authorization tidak ditemukan.",
		})
	}

	authHeaderParts := strings.Split(rawAuthHeader, " ")
	if len(authHeaderParts) != 2 || authHeaderParts[0] != "Bearer" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Format token tidak valid. Gunakan format 'Bearer <token>'.",
		})
	}

	rawBearerToken := authHeaderParts[1]
	claims, err := utils.ValidateUserToken(rawBearerToken)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Token tidak valid atau telah kedaluwarsa.",
			Error:   err.Error(),
		})
	}

	ctx.Locals("user_id", claims.UserID)
	ctx.Locals("user_email", claims.Email)
	ctx.Locals("user_role", claims.Role)

	return ctx.Next()
}
