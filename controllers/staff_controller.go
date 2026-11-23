package controllers

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

var validStaffUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,50}$`)

type StaffController struct {
	repo repositories.AdminUserRepository
}

func NewStaffController(repo repositories.AdminUserRepository) *StaffController {
	return &StaffController{repo: repo}
}

func isValidStaffRole(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "SUPER_ADMIN", "ADMIN", "PETUGAS":
		return true
	default:
		return false
	}
}

func resolveActorFromCtx(ctx *fiber.Ctx) (string, string, string) {
	actorID, _ := ctx.Locals("admin_id").(string)
	actorEmail, _ := ctx.Locals("admin_email").(string)
	actorRole, _ := ctx.Locals("admin_role").(string)
	if actorID == "" {
		actorID = "system"
	}
	if actorEmail == "" {
		actorEmail = "system@symphoniatic.id"
	}
	if actorRole == "" {
		actorRole = "SUPER_ADMIN"
	}
	return actorID, actorEmail, actorRole
}

func (c *StaffController) recordAudit(ctx *fiber.Ctx, action, targetResource, details string) {
	if c.repo == nil {
		return
	}
	actorID, actorEmail, actorRole := resolveActorFromCtx(ctx)
	_ = c.repo.CreateAuditLog(ctx.Context(), &models.AuditLog{
		ActorID:        actorID,
		ActorName:      actorEmail,
		ActorRole:      actorRole,
		Action:         action,
		TargetResource: targetResource,
		Details:        details,
		IPAddress:      ctx.IP(),
	})
}

func (c *StaffController) GetAllStaff(ctx *fiber.Ctx) error {
	roleFilter := ctx.Query("role")
	search := ctx.Query("search")

	staff, err := c.repo.GetAllStaff(ctx.Context(), roleFilter, search)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil daftar akun staf", err)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Daftar akun staf berhasil diambil",
		Data:    staff,
	})
}

func (c *StaffController) CreateStaff(ctx *fiber.Ctx) error {
	var req models.CreateStaffRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format payload pembuatan akun staf tidak valid",
			Error:   err.Error(),
		})
	}

	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.ToUpper(strings.TrimSpace(req.Role))

	if !validStaffUsernameRegex.MatchString(req.Username) {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Username harus 3-50 karakter dan hanya boleh berisi huruf, angka, titik, garis bawah, atau tanda hubung",
		})
	}
	if req.Name == "" || len(req.Name) < 2 {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Nama lengkap staf wajib diisi minimal 2 karakter",
		})
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format alamat email staf tidak valid",
		})
	}
	if len(req.Password) < 6 {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kata sandi awal staf minimal 6 karakter",
		})
	}
	if !isValidStaffRole(req.Role) {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Peran staf tidak valid. Pilih SUPER_ADMIN, ADMIN, atau PETUGAS",
		})
	}

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengamankan kata sandi staf", err)
	}

	created, err := c.repo.CreateStaff(ctx.Context(), &req, passwordHash)
	if err != nil {
		if errors.Is(err, repositories.ErrStaffDuplicateUsername) {
			return ctx.Status(fiber.StatusConflict).JSON(models.APIResponse{
				Success: false,
				Message: err.Error(),
			})
		}
		return utils.ResponseInternalError(ctx, "Gagal membuat akun staf baru", err)
	}

	c.recordAudit(
		ctx,
		"CREATE_STAFF",
		"staff:"+created.Username,
		fmt.Sprintf("Membuat akun staf %s (%s) dengan peran %s", created.Name, created.Email, created.Role),
	)

	return ctx.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Akun staf baru berhasil ditambahkan",
		Data:    created,
	})
}

func (c *StaffController) UpdateStaff(ctx *fiber.Ctx) error {
	targetID := strings.TrimSpace(ctx.Params("id"))
	if targetID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "ID staf tidak valid",
		})
	}

	var req models.UpdateStaffRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format payload pembaruan staf tidak valid",
			Error:   err.Error(),
		})
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.ToUpper(strings.TrimSpace(req.Role))

	if req.Name == "" || len(req.Name) < 2 {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Nama lengkap staf wajib diisi minimal 2 karakter",
		})
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format alamat email staf tidak valid",
		})
	}
	if !isValidStaffRole(req.Role) {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Peran staf tidak valid. Pilih SUPER_ADMIN, ADMIN, atau PETUGAS",
		})
	}

	existing, err := c.repo.GetByID(ctx.Context(), targetID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Akun staf tidak ditemukan",
			})
		}
		return utils.ResponseInternalError(ctx, "Gagal memeriksa data akun staf", err)
	}

	actorID, _, _ := resolveActorFromCtx(ctx)
	if existing.ID == actorID && req.Role != "SUPER_ADMIN" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: repositories.ErrStaffSelfDeactivate.Error(),
		})
	}
	if existing.Role == "SUPER_ADMIN" && existing.IsActive && req.Role != "SUPER_ADMIN" {
		count, countErr := c.repo.CountActiveSuperAdmins(ctx.Context())
		if countErr != nil {
			return utils.ResponseInternalError(ctx, "Gagal memeriksa jumlah Super Admin aktif", countErr)
		}
		if count <= 1 {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: repositories.ErrStaffLastSuperAdmin.Error(),
			})
		}
	}

	updated, err := c.repo.UpdateStaff(ctx.Context(), targetID, &req)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Akun staf tidak ditemukan",
			})
		}
		if errors.Is(err, repositories.ErrStaffDuplicateUsername) {
			return ctx.Status(fiber.StatusConflict).JSON(models.APIResponse{
				Success: false,
				Message: err.Error(),
			})
		}
		return utils.ResponseInternalError(ctx, "Gagal memperbarui akun staf", err)
	}

	c.recordAudit(
		ctx,
		"UPDATE_STAFF",
		"staff:"+updated.Username,
		fmt.Sprintf("Memperbarui profil staf %s (role %s -> %s)", updated.Username, existing.Role, updated.Role),
	)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Data akun staf berhasil diperbarui",
		Data:    updated,
	})
}

func (c *StaffController) ToggleStaffStatus(ctx *fiber.Ctx) error {
	targetID := strings.TrimSpace(ctx.Params("id"))
	if targetID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "ID staf tidak valid",
		})
	}

	existing, err := c.repo.GetByID(ctx.Context(), targetID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Akun staf tidak ditemukan",
			})
		}
		return utils.ResponseInternalError(ctx, "Gagal memeriksa status akun staf", err)
	}

	actorID, _, _ := resolveActorFromCtx(ctx)
	if existing.ID == actorID && existing.IsActive {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: repositories.ErrStaffSelfDeactivate.Error(),
		})
	}

	if existing.Role == "SUPER_ADMIN" && existing.IsActive {
		count, countErr := c.repo.CountActiveSuperAdmins(ctx.Context())
		if countErr != nil {
			return utils.ResponseInternalError(ctx, "Gagal memeriksa jumlah Super Admin aktif", countErr)
		}
		if count <= 1 {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: repositories.ErrStaffLastSuperAdmin.Error(),
			})
		}
	}

	updated, err := c.repo.ToggleStaffActive(ctx.Context(), targetID)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengubah status aktif akun staf", err)
	}

	statusLabel := "NONAKTIF"
	if updated.IsActive {
		statusLabel = "AKTIF"
	}

	c.recordAudit(
		ctx,
		"TOGGLE_STAFF_STATUS",
		"staff:"+updated.Username,
		fmt.Sprintf("Mengubah status akun staf %s menjadi %s", updated.Username, statusLabel),
	)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Status akun %s berhasil diubah menjadi %s", updated.Username, statusLabel),
		Data:    updated,
	})
}

func (c *StaffController) ResetStaffPassword(ctx *fiber.Ctx) error {
	targetID := strings.TrimSpace(ctx.Params("id"))
	if targetID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "ID staf tidak valid",
		})
	}

	var req models.ResetStaffPasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format payload reset kata sandi tidak valid",
			Error:   err.Error(),
		})
	}

	if len(req.NewPassword) < 6 {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kata sandi baru minimal harus 6 karakter",
		})
	}

	existing, err := c.repo.GetByID(ctx.Context(), targetID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Akun staf tidak ditemukan",
			})
		}
		return utils.ResponseInternalError(ctx, "Gagal memeriksa akun staf", err)
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengamankan kata sandi baru", err)
	}

	if err := c.repo.ResetStaffPassword(ctx.Context(), targetID, hash); err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mereset kata sandi staf", err)
	}

	c.recordAudit(
		ctx,
		"RESET_STAFF_PASSWORD",
		"staff:"+existing.Username,
		fmt.Sprintf("Mereset kata sandi akun staf %s (%s)", existing.Name, existing.Role),
	)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Kata sandi akun %s berhasil diatur ulang", existing.Username),
	})
}

func (c *StaffController) DeleteStaff(ctx *fiber.Ctx) error {
	targetID := strings.TrimSpace(ctx.Params("id"))
	if targetID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "ID staf tidak valid",
		})
	}

	existing, err := c.repo.GetByID(ctx.Context(), targetID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Akun staf tidak ditemukan",
			})
		}
		return utils.ResponseInternalError(ctx, "Gagal memeriksa akun staf", err)
	}

	actorID, _, _ := resolveActorFromCtx(ctx)
	if existing.ID == actorID {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: repositories.ErrStaffSelfDeactivate.Error(),
		})
	}

	if existing.Role == "SUPER_ADMIN" && existing.IsActive {
		count, countErr := c.repo.CountActiveSuperAdmins(ctx.Context())
		if countErr != nil {
			return utils.ResponseInternalError(ctx, "Gagal memeriksa jumlah Super Admin aktif", countErr)
		}
		if count <= 1 {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: repositories.ErrStaffLastSuperAdmin.Error(),
			})
		}
	}

	if err := c.repo.DeleteStaff(ctx.Context(), targetID); err != nil {
		return utils.ResponseInternalError(ctx, "Gagal menghapus akun staf", err)
	}

	c.recordAudit(
		ctx,
		"DELETE_STAFF",
		"staff:"+existing.Username,
		fmt.Sprintf("Menghapus permanen akun staf %s (%s)", existing.Name, existing.Role),
	)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Akun staf %s berhasil dihapus", existing.Username),
	})
}

func (c *StaffController) GetAuditLogs(ctx *fiber.Ctx) error {
	actionFilter := ctx.Query("action")
	search := ctx.Query("search")
	limit := 200
	if rawLimit := strings.TrimSpace(ctx.Query("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := c.repo.GetAuditLogs(ctx.Context(), actionFilter, search, limit)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil riwayat log audit", err)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Riwayat log audit berhasil diambil",
		Data:    logs,
	})
}

func classifyAdminAuditAction(method, path string) (string, string) {
	switch {
	case strings.Contains(path, "/events") && strings.Contains(path, "/categories"):
		return "MANAGE_CATEGORY", "category"
	case strings.Contains(path, "/categories"):
		return "MANAGE_CATEGORY", "category"
	case strings.Contains(path, "/events") && strings.Contains(path, "/toggle-close"):
		return "TOGGLE_EVENT_SALES", "event"
	case strings.Contains(path, "/events") && method == fiber.MethodPost:
		return "CREATE_EVENT", "event"
	case strings.Contains(path, "/events") && method == fiber.MethodPut:
		return "UPDATE_EVENT", "event"
	case strings.Contains(path, "/events") && method == fiber.MethodDelete:
		return "DELETE_EVENT", "event"
	case strings.Contains(path, "/orders") && strings.Contains(path, "/status"):
		return "UPDATE_ORDER_STATUS", "order"
	case strings.Contains(path, "/tickets/checkin"):
		return "TICKET_CHECKIN", "ticket"
	case strings.Contains(path, "/refunds") && strings.Contains(path, "/status"):
		return "UPDATE_REFUND_STATUS", "refund"
	case strings.Contains(path, "/promos") && strings.Contains(path, "/toggle-active"):
		return "TOGGLE_PROMO_STATUS", "promo"
	case strings.Contains(path, "/promos") && method == fiber.MethodPost:
		return "CREATE_PROMO", "promo"
	case strings.Contains(path, "/promos") && method == fiber.MethodPut:
		return "UPDATE_PROMO", "promo"
	case strings.Contains(path, "/promos") && method == fiber.MethodDelete:
		return "DELETE_PROMO", "promo"
	case strings.Contains(path, "/upload"):
		return "UPLOAD_MEDIA", "media"
	default:
		return method + "_ADMIN_ACTION", "admin"
	}
}

// AuditTrailMiddleware mencatat setiap perubahan state operasional admin/petugas yang berhasil ke PostgreSQL.
func AuditTrailMiddleware(staffRepo repositories.AdminUserRepository) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		err := ctx.Next()
		if staffRepo == nil {
			return err
		}

		method := ctx.Method()
		if method != fiber.MethodPost && method != fiber.MethodPut && method != fiber.MethodPatch && method != fiber.MethodDelete {
			return err
		}

		path := ctx.Path()
		if strings.HasSuffix(path, "/admin/login") || strings.Contains(path, "/admin/staff") {
			return err
		}

		if err == nil && ctx.Response().StatusCode() >= 200 && ctx.Response().StatusCode() < 400 {
			actorID, actorEmail, actorRole := resolveActorFromCtx(ctx)
			if actorID != "system" {
				action, resourceKind := classifyAdminAuditAction(method, path)
				_ = staffRepo.CreateAuditLog(ctx.Context(), &models.AuditLog{
					ActorID:        actorID,
					ActorName:      actorEmail,
					ActorRole:      actorRole,
					Action:         action,
					TargetResource: resourceKind + ":" + path,
					Details:        fmt.Sprintf("%s %s (HTTP %d)", method, path, ctx.Response().StatusCode()),
					IPAddress:      ctx.IP(),
				})
			}
		}

		return err
	}
}
