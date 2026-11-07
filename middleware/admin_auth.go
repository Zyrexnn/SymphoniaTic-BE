package middleware

import (
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

// RequireAdminAuth membatasi akses endpoint khusus untuk peran administratif tertentu.
// Peran default adalah ADMIN apabila parameter allowedRoles tidak didefinisikan.
func RequireAdminAuth(allowedRoles ...string) fiber.Handler {
	targetRoles := allowedRoles
	if len(targetRoles) == 0 {
		targetRoles = []string{"ADMIN"}
	}

	return func(ctx *fiber.Ctx) error {
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

		roleAllowed := false
		for _, role := range targetRoles {
			if claims.Role == role {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			return ctx.Status(fiber.StatusForbidden).JSON(models.APIResponse{
				Success: false,
				Message: "Akses ditolak. Peran pengguna tidak memiliki hak akses yang memadai.",
			})
		}

		ctx.Locals("admin_id", claims.UserID)
		ctx.Locals("admin_email", claims.Email)
		ctx.Locals("admin_role", claims.Role)

		return ctx.Next()
	}
}
