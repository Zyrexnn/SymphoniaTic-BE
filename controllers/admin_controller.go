package controllers

import (
	"crypto/sha256"
	"crypto/subtle"
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
	repo      repositories.OrderRepository
	staffRepo repositories.AdminUserRepository
	authCfg   *config.AuthConfig
}

func NewAdminController(repo repositories.OrderRepository, authCfg *config.AuthConfig) *AdminController {
	return &AdminController{
		repo:    repo,
		authCfg: authCfg,
	}
}

func NewAdminControllerWithStaffRepo(repo repositories.OrderRepository, staffRepo repositories.AdminUserRepository, authCfg *config.AuthConfig) *AdminController {
	return &AdminController{
		repo:      repo,
		staffRepo: staffRepo,
		authCfg:   authCfg,
	}
}

func constantTimeMatch(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
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

	cleanUsername := strings.TrimSpace(adminLoginReq.Username)
	if cleanUsername == "" || adminLoginReq.Password == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Username atau Password salah",
		})
	}

	var authenticatedRole string
	var authenticatedName string
	var authenticatedID string
	var authenticatedEmail string

	// Cek akun staf dari PostgreSQL terlebih dahulu bila repository tersedia
	if c.staffRepo != nil {
		staffUser, err := c.staffRepo.GetByUsername(ctx.Context(), cleanUsername)
		if err == nil && staffUser != nil {
			if !staffUser.IsActive {
				return ctx.Status(fiber.StatusForbidden).JSON(models.APIResponse{
					Success: false,
					Message: "Akun staf ini sedang dinonaktifkan. Hubungi Super Admin.",
				})
			}
			if !utils.CheckPasswordHash(adminLoginReq.Password, staffUser.PasswordHash) {
				return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
					Success: false,
					Message: "Username atau Password salah",
				})
			}
			authenticatedRole = staffUser.Role
			authenticatedName = staffUser.Name
			authenticatedID = staffUser.ID
			authenticatedEmail = staffUser.Email
		}
	}

	if authenticatedID == "" {
		var superUser, superPass, adminUser, adminPass, petugasUser, petugasPass string

		if c.authCfg != nil {
			superUser = c.authCfg.SuperAdminUsername
			superPass = c.authCfg.SuperAdminPassword
			adminUser = c.authCfg.AdminUsername
			adminPass = c.authCfg.AdminPassword
			petugasUser = c.authCfg.PetugasUsername
			petugasPass = c.authCfg.PetugasPassword
		} else {
			superUser = os.Getenv("SUPER_ADMIN_USERNAME")
			superPass = os.Getenv("SUPER_ADMIN_PASSWORD")
			adminUser = os.Getenv("ADMIN_USERNAME")
			adminPass = os.Getenv("ADMIN_PASSWORD")
			petugasUser = os.Getenv("PETUGAS_USERNAME")
			petugasPass = os.Getenv("PETUGAS_PASSWORD")
		}

		if adminUser == "" && superUser == "" {
			return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
				Success: false,
				Message: "Autentikasi admin belum dikonfigurasi di environment server",
			})
		}

		superUserMatch := superUser != "" && superPass != "" && constantTimeMatch(cleanUsername, superUser)
		superPassMatch := superUser != "" && superPass != "" && constantTimeMatch(adminLoginReq.Password, superPass)
		adminUserMatch := adminUser != "" && adminPass != "" && constantTimeMatch(cleanUsername, adminUser)
		adminPassMatch := adminUser != "" && adminPass != "" && constantTimeMatch(adminLoginReq.Password, adminPass)
		petugasUserMatch := petugasUser != "" && petugasPass != "" && constantTimeMatch(cleanUsername, petugasUser)
		petugasPassMatch := petugasUser != "" && petugasPass != "" && constantTimeMatch(adminLoginReq.Password, petugasPass)

		if superUserMatch && superPassMatch {
			authenticatedRole = "SUPER_ADMIN"
			authenticatedName = "Super Administrator"
			authenticatedID = "staff-superadmin-1"
			authenticatedEmail = cleanUsername + "@symphoniatic.id"
		} else if adminUserMatch && adminPassMatch {
			authenticatedRole = "ADMIN"
			authenticatedName = "Administrator Operasional"
			authenticatedID = "admin-1"
			authenticatedEmail = cleanUsername + "@symphoniatic.id"
		} else if petugasUserMatch && petugasPassMatch {
			authenticatedRole = "PETUGAS"
			authenticatedName = "Petugas Gate Officer"
			authenticatedID = "petugas-1"
			authenticatedEmail = cleanUsername + "@symphoniatic.id"
		} else {
			return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
				Success: false,
				Message: "Username atau Password salah",
			})
		}
	}

	adminJwtToken, err := utils.GenerateUserToken(authenticatedID, authenticatedEmail, authenticatedRole)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal menerbitkan token sesi", err)
	}

	if c.staffRepo != nil {
		_ = c.staffRepo.CreateAuditLog(ctx.Context(), &models.AuditLog{
			ActorID:        authenticatedID,
			ActorName:      authenticatedName,
			ActorRole:      authenticatedRole,
			Action:         "ADMIN_LOGIN",
			TargetResource: "auth:" + cleanUsername,
			Details:        fmt.Sprintf("Login portal operasional sebagai %s", authenticatedRole),
			IPAddress:      ctx.IP(),
			Category:       "AUTH",
			Severity:       "INFO",
			StatusCode:     fiber.StatusOK,
			UserAgent:      ctx.Get("User-Agent"),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Login %s berhasil", authenticatedRole),
		Data: models.AuthResponseData{
			Token: adminJwtToken,
			User: models.UserRecord{
				ID:         authenticatedID,
				Email:      authenticatedEmail,
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

	const maxFileSize = 5 * 1024 * 1024
	if uploadedFile.Size > maxFileSize {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Ukuran file terlalu besar. Maksimal ukuran gambar adalah 5MB.",
		})
	}

	// Validasi whitelist ekstensi dan tipe konten untuk mencegah eksekusi skrip berbahaya
	fileExtension := strings.ToLower(filepath.Ext(uploadedFile.Filename))
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	if !allowedExtensions[fileExtension] {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format file tidak didukung. Harap unggah gambar JPG, JPEG, PNG, atau WEBP.",
		})
	}

	contentType := strings.ToLower(uploadedFile.Header.Get("Content-Type"))
	if contentType != "" && contentType != "application/octet-stream" && !strings.HasPrefix(contentType, "image/") {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tipe konten file tidak valid. Harap unggah berkas gambar yang sah.",
		})
	}

	uploadsDir := filepath.Clean("uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat folder penyimpanan",
			Error:   err.Error(),
		})
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExtension)
	savePath := filepath.Clean(filepath.Join(uploadsDir, filename))

	if err := ctx.SaveFile(uploadedFile, savePath); err != nil {
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

	// URL publik selalu menggunakan forward slash '/' untuk konsistensi HTTP pada sistem operasi Windows, Linux, dan macOS
	publicURL := fmt.Sprintf("%s/uploads/%s", strings.TrimRight(baseURL, "/"), filename)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "File gambar berhasil diunggah",
		Data: fiber.Map{
			"url": publicURL,
		},
	})
}
