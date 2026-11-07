package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

type AdminController struct {
	repo    repositories.OrderRepository
	authCfg *config.AuthConfig
}

func NewAdminController(repo repositories.OrderRepository, authCfg *config.AuthConfig) *AdminController {
	return &AdminController{
		repo:    repo,
		authCfg: authCfg,
	}
}

// AdminLogin godoc
// @Summary Login Administrator
// @Description Autentikasi khusus akun administrator untuk mengakses panel manajemen admin.
// @Tags Auth - Admin
// @Accept json
// @Produce json
// @Param payload body models.AdminLoginRequest true "Admin Login Payload"
// @Success 200 {object} models.APIResponse{data=models.AuthResponseData} "Login Admin berhasil"
// @Failure 401 {object} models.APIResponse "Username atau Password Admin salah"
// @Router /admin/login [post]
func (c *AdminController) AdminLogin(ctx *fiber.Ctx) error {
	var adminLoginReq models.AdminLoginRequest
	if err := ctx.BodyParser(&adminLoginReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request login tidak valid",
			Error:   err.Error(),
		})
	}

	adminUser := "admin"
	adminPass := "123"
	petugasUser := "petugas"
	petugasPass := "123"

	if c.authCfg != nil {
		if c.authCfg.AdminUsername != "" {
			adminUser = c.authCfg.AdminUsername
		}
		if c.authCfg.AdminPassword != "" {
			adminPass = c.authCfg.AdminPassword
		}
		if c.authCfg.PetugasUsername != "" {
			petugasUser = c.authCfg.PetugasUsername
		}
		if c.authCfg.PetugasPassword != "" {
			petugasPass = c.authCfg.PetugasPassword
		}
	} else {
		if u := os.Getenv("ADMIN_USERNAME"); u != "" {
			adminUser = u
		}
		if p := os.Getenv("ADMIN_PASSWORD"); p != "" {
			adminPass = p
		}
		if u := os.Getenv("PETUGAS_USERNAME"); u != "" {
			petugasUser = u
		}
		if p := os.Getenv("PETUGAS_PASSWORD"); p != "" {
			petugasPass = p
		}
	}

	var authenticatedRole string
	var authenticatedName string
	var authenticatedID string

	if adminLoginReq.Username == adminUser && adminLoginReq.Password == adminPass {
		authenticatedRole = "ADMIN"
		authenticatedName = "Super Administrator"
		authenticatedID = "admin-1"
	} else if adminLoginReq.Username == petugasUser && adminLoginReq.Password == petugasPass {
		authenticatedRole = "PETUGAS"
		authenticatedName = "Petugas Gate Officer"
		authenticatedID = "petugas-1"
	} else {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Username atau Password salah",
		})
	}

	adminJwtToken, err := utils.GenerateUserToken(authenticatedID, adminLoginReq.Username+"@symphoniatic.id", authenticatedRole)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal menerbitkan token sesi", err)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Login %s berhasil", authenticatedRole),
		Data: models.AuthResponseData{
			Token: adminJwtToken,
			User: models.UserRecord{
				ID:         authenticatedID,
				Email:      adminLoginReq.Username + "@symphoniatic.id",
				Name:       authenticatedName,
				Role:       authenticatedRole,
				IsVerified: true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		},
	})
}

// GetAdminDashboardMetrics godoc
// @Summary Mengambil ringkasan metrik dashboard admin
// @Description Mengagregasi metrik analitik bisnis (total pendapatan, tiket terjual, sisa kuota, timeline bulanan).
// @Tags Admin - Analytics
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.AdminDashboardMetrics} "Metrik admin berhasil diambil"
// @Router /admin/dashboard [get]
func (c *AdminController) GetAdminDashboardMetrics(ctx *fiber.Ctx) error {
	metrics, err := c.repo.GetAdminDashboardMetrics(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil metrik dashboard admin",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Metrik admin berhasil diambil",
		Data:    metrics,
	})
}

// UploadImage godoc
// @Summary Mengunggah gambar poster/banner konser (Admin)
// @Description Menyimpan file gambar ke direktori server lokal dan mengembalikan URL publik.
// @Tags Admin - Events Management
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "File gambar poster (jpg/png/webp)"
// @Success 200 {object} models.APIResponse "File gambar berhasil diunggah"
// @Failure 400 {object} models.APIResponse "Gagal membaca file gambar"
// @Router /admin/upload [post]
func (c *AdminController) UploadImage(ctx *fiber.Ctx) error {
	uploadedFile, err := ctx.FormFile("image")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membaca file gambar",
			Error:   err.Error(),
		})
	}

	uploadDir := filepath.Clean("uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat folder penyimpanan",
			Error:   err.Error(),
		})
	}

	fileExtension := filepath.Ext(uploadedFile.Filename)
	uniqueFilename := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExtension)
	uploadDestPath := filepath.Clean(filepath.Join(uploadDir, uniqueFilename))

	if err := ctx.SaveFile(uploadedFile, uploadDestPath); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menyimpan file gambar",
			Error:   err.Error(),
		})
	}

	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8082"
	} else {
		baseURL = strings.TrimSuffix(baseURL, "/api/v1")
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "File gambar berhasil diunggah",
		Data: fiber.Map{
			"url": fmt.Sprintf("%s/uploads/%s", baseURL, uniqueFilename),
		},
	})
}
