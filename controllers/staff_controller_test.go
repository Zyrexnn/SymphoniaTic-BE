package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

func setupStaffControllerTestApp(repo repositories.AdminUserRepository) *fiber.App {
	app := fiber.New()
	ctrl := controllers.NewStaffController(repo)

	guard := middleware.RequireAdminAuth("SUPER_ADMIN")
	api := app.Group("/api/v1/admin")
	api.Get("/staff", guard, ctrl.GetAllStaff)
	api.Post("/staff", guard, ctrl.CreateStaff)
	api.Put("/staff/:id", guard, ctrl.UpdateStaff)
	api.Patch("/staff/:id/status", guard, ctrl.ToggleStaffStatus)
	api.Post("/staff/:id/reset-password", guard, ctrl.ResetStaffPassword)
	api.Delete("/staff/:id", guard, ctrl.DeleteStaff)
	api.Get("/audit-logs", guard, ctrl.GetAuditLogs)

	return app
}

func TestStaffController_CRUDAndGuards(t *testing.T) {
	t.Setenv("JWT_SECRET", "symphoniatic-test-jwt-secret-key-min-32")

	superToken, err := utils.GenerateUserToken("staff-super-1", "superadmin@symphoniatic.id", "SUPER_ADMIN")
	if err != nil {
		t.Fatalf("Gagal membuat token SUPER_ADMIN: %v", err)
	}
	adminToken, err := utils.GenerateUserToken("staff-admin-1", "admin@symphoniatic.id", "ADMIN")
	if err != nil {
		t.Fatalf("Gagal membuat token ADMIN: %v", err)
	}

	t.Run("Menolak akses manajemen staf bagi role ADMIN (403 Forbidden)", func(t *testing.T) {
		app := setupStaffControllerTestApp(&repositories.MockAdminUserRepository{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/staff", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal kirim request: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Ekspektasi 403 Forbidden untuk ADMIN, dapat %d", resp.StatusCode)
		}
	})

	t.Run("SUPER_ADMIN berhasil membuat akun staf baru dan mencatat audit log", func(t *testing.T) {
		var auditRecorded bool
		repo := &repositories.MockAdminUserRepository{
			CreateStaffFunc: func(ctx context.Context, req *models.CreateStaffRequest, passwordHash string) (*models.AdminUser, error) {
				if !utils.CheckPasswordHash(req.Password, passwordHash) {
					t.Errorf("Password hash tidak cocok dengan kata sandi plaintext")
				}
				return &models.AdminUser{
					ID:        "staff-new-1",
					Username:  req.Username,
					Name:      req.Name,
					Email:     req.Email,
					Role:      req.Role,
					IsActive:  true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
			CreateAuditLogFunc: func(ctx context.Context, logEntry *models.AuditLog) error {
				if logEntry.Action == "CREATE_STAFF" && logEntry.TargetResource == "staff:gate_budi" {
					auditRecorded = true
				}
				return nil
			},
		}

		app := setupStaffControllerTestApp(repo)
		body, _ := json.Marshal(models.CreateStaffRequest{
			Username: "gate_budi",
			Name:     "Budi Santoso",
			Email:    "budi.gate@symphoniatic.id",
			Password: "Password123!",
			Role:     "PETUGAS",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/staff", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal kirim request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Ekspektasi 201 Created, dapat %d", resp.StatusCode)
		}
		if !auditRecorded {
			t.Fatal("Pembuatan staf wajib mencatat audit log CREATE_STAFF")
		}
	})

	t.Run("UpdateStaff berhasil mengubah profil & peran serta mencegah downgrade diri sendiri atau Super Admin terakhir", func(t *testing.T) {
		var updateAuditRecorded bool
		repo := &repositories.MockAdminUserRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*models.AdminUser, error) {
				switch id {
				case "staff-super-1":
					return &models.AdminUser{ID: "staff-super-1", Username: "superadmin", Role: "SUPER_ADMIN", IsActive: true}, nil
				case "staff-super-2":
					return &models.AdminUser{ID: "staff-super-2", Username: "superadmin2", Role: "SUPER_ADMIN", IsActive: true}, nil
				case "staff-petugas-1":
					return &models.AdminUser{ID: "staff-petugas-1", Username: "petugas1", Name: "Petugas 1", Email: "p1@symphoniatic.id", Role: "PETUGAS", IsActive: true}, nil
				default:
					return nil, repositories.ErrNotFound
				}
			},
			CountActiveSuperAdminsFunc: func(ctx context.Context) (int, error) {
				return 1, nil
			},
			UpdateStaffFunc: func(ctx context.Context, id string, req *models.UpdateStaffRequest) (*models.AdminUser, error) {
				if req.Email == "conflict@symphoniatic.id" {
					return nil, repositories.ErrStaffDuplicateUsername
				}
				return &models.AdminUser{
					ID:        id,
					Username:  "petugas1",
					Name:      req.Name,
					Email:     req.Email,
					Role:      req.Role,
					IsActive:  true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
			CreateAuditLogFunc: func(ctx context.Context, logEntry *models.AuditLog) error {
				if logEntry.Action == "UPDATE_STAFF" {
					updateAuditRecorded = true
				}
				return nil
			},
		}
		app := setupStaffControllerTestApp(repo)

		// 1. Self role downgrade ditolak (400)
		selfBody, _ := json.Marshal(models.UpdateStaffRequest{Name: "Self", Email: "self@symphoniatic.id", Role: "ADMIN"})
		reqSelf := httptest.NewRequest(http.MethodPut, "/api/v1/admin/staff/staff-super-1", bytes.NewReader(selfBody))
		reqSelf.Header.Set("Authorization", "Bearer "+superToken)
		reqSelf.Header.Set("Content-Type", "application/json")
		respSelf, _ := app.Test(reqSelf, -1)
		if respSelf.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 saat menurunkan role sendiri, dapat %d", respSelf.StatusCode)
		}

		// 2. Last active SUPER_ADMIN downgrade ditolak (400)
		lastBody, _ := json.Marshal(models.UpdateStaffRequest{Name: "Super 2", Email: "s2@symphoniatic.id", Role: "ADMIN"})
		reqLast := httptest.NewRequest(http.MethodPut, "/api/v1/admin/staff/staff-super-2", bytes.NewReader(lastBody))
		reqLast.Header.Set("Authorization", "Bearer "+superToken)
		reqLast.Header.Set("Content-Type", "application/json")
		respLast, _ := app.Test(reqLast, -1)
		if respLast.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 saat menurunkan Super Admin terakhir, dapat %d", respLast.StatusCode)
		}

		// 3. Duplicate email -> 409 Conflict
		conflictBody, _ := json.Marshal(models.UpdateStaffRequest{Name: "Petugas 1", Email: "conflict@symphoniatic.id", Role: "ADMIN"})
		reqConflict := httptest.NewRequest(http.MethodPut, "/api/v1/admin/staff/staff-petugas-1", bytes.NewReader(conflictBody))
		reqConflict.Header.Set("Authorization", "Bearer "+superToken)
		reqConflict.Header.Set("Content-Type", "application/json")
		respConflict, _ := app.Test(reqConflict, -1)
		if respConflict.StatusCode != http.StatusConflict {
			t.Fatalf("Ekspektasi 409 Conflict saat email duplikat, dapat %d", respConflict.StatusCode)
		}

		// 4. Missing ID -> 404 Not Found
		reqNotFound := httptest.NewRequest(http.MethodPut, "/api/v1/admin/staff/staff-missing", bytes.NewReader(lastBody))
		reqNotFound.Header.Set("Authorization", "Bearer "+superToken)
		reqNotFound.Header.Set("Content-Type", "application/json")
		respNotFound, _ := app.Test(reqNotFound, -1)
		if respNotFound.StatusCode != http.StatusNotFound {
			t.Fatalf("Ekspektasi 404 Not Found, dapat %d", respNotFound.StatusCode)
		}

		// 5. Sukses update + audit log
		okBody, _ := json.Marshal(models.UpdateStaffRequest{Name: "Petugas Senior", Email: "senior@symphoniatic.id", Role: "ADMIN"})
		reqOK := httptest.NewRequest(http.MethodPut, "/api/v1/admin/staff/staff-petugas-1", bytes.NewReader(okBody))
		reqOK.Header.Set("Authorization", "Bearer "+superToken)
		reqOK.Header.Set("Content-Type", "application/json")
		respOK, _ := app.Test(reqOK, -1)
		if respOK.StatusCode != http.StatusOK || !updateAuditRecorded {
			t.Fatalf("UpdateStaff gagal: status=%d audit=%v", respOK.StatusCode, updateAuditRecorded)
		}
	})

	t.Run("Mencegah Super Admin menonaktifkan atau menghapus akun dirinya sendiri", func(t *testing.T) {
		repo := &repositories.MockAdminUserRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*models.AdminUser, error) {
				return &models.AdminUser{
					ID:       "staff-super-1",
					Username: "superadmin",
					Role:     "SUPER_ADMIN",
					IsActive: true,
				}, nil
			},
		}
		app := setupStaffControllerTestApp(repo)

		reqToggle := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/staff/staff-super-1/status", nil)
		reqToggle.Header.Set("Authorization", "Bearer "+superToken)
		respToggle, _ := app.Test(reqToggle, -1)
		if respToggle.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request saat menonaktifkan diri sendiri, dapat %d", respToggle.StatusCode)
		}

		reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/staff/staff-super-1", nil)
		reqDel.Header.Set("Authorization", "Bearer "+superToken)
		respDel, _ := app.Test(reqDel, -1)
		if respDel.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request saat menghapus diri sendiri, dapat %d", respDel.StatusCode)
		}
	})

	t.Run("ToggleStaffStatus dan DeleteStaff berfungsi untuk staf lain dan menolak penghapusan Super Admin terakhir", func(t *testing.T) {
		var toggleAudit, deleteAudit bool
		repo := &repositories.MockAdminUserRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*models.AdminUser, error) {
				if id == "staff-super-last" {
					return &models.AdminUser{ID: id, Username: "lastsuper", Role: "SUPER_ADMIN", IsActive: true}, nil
				}
				if id == "staff-petugas-2" {
					return &models.AdminUser{ID: id, Username: "petugas2", Name: "Petugas 2", Role: "PETUGAS", IsActive: true}, nil
				}
				return nil, repositories.ErrNotFound
			},
			CountActiveSuperAdminsFunc: func(ctx context.Context) (int, error) {
				return 1, nil
			},
			ToggleStaffActiveFunc: func(ctx context.Context, id string) (*models.AdminUser, error) {
				return &models.AdminUser{ID: id, Username: "petugas2", Role: "PETUGAS", IsActive: false}, nil
			},
			DeleteStaffFunc: func(ctx context.Context, id string) error {
				return nil
			},
			CreateAuditLogFunc: func(ctx context.Context, logEntry *models.AuditLog) error {
				if logEntry.Action == "TOGGLE_STAFF_STATUS" {
					toggleAudit = true
				}
				if logEntry.Action == "DELETE_STAFF" {
					deleteAudit = true
				}
				return nil
			},
		}
		app := setupStaffControllerTestApp(repo)

		// Toggle off Super Admin terakhir -> 400
		reqLastToggle := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/staff/staff-super-last/status", nil)
		reqLastToggle.Header.Set("Authorization", "Bearer "+superToken)
		respLastToggle, _ := app.Test(reqLastToggle, -1)
		if respLastToggle.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 saat menonaktifkan Super Admin terakhir, dapat %d", respLastToggle.StatusCode)
		}

		// Delete Super Admin terakhir -> 400
		reqLastDel := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/staff/staff-super-last", nil)
		reqLastDel.Header.Set("Authorization", "Bearer "+superToken)
		respLastDel, _ := app.Test(reqLastDel, -1)
		if respLastDel.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 saat menghapus Super Admin terakhir, dapat %d", respLastDel.StatusCode)
		}

		// Toggle off petugas2 -> 200
		reqToggle := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/staff/staff-petugas-2/status", nil)
		reqToggle.Header.Set("Authorization", "Bearer "+superToken)
		respToggle, _ := app.Test(reqToggle, -1)
		if respToggle.StatusCode != http.StatusOK || !toggleAudit {
			t.Fatalf("ToggleStaffStatus gagal: status=%d audit=%v", respToggle.StatusCode, toggleAudit)
		}

		// Delete petugas2 -> 200
		reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/staff/staff-petugas-2", nil)
		reqDel.Header.Set("Authorization", "Bearer "+superToken)
		respDel, _ := app.Test(reqDel, -1)
		if respDel.StatusCode != http.StatusOK || !deleteAudit {
			t.Fatalf("DeleteStaff gagal: status=%d audit=%v", respDel.StatusCode, deleteAudit)
		}
	})

	t.Run("Reset password dan pengambilan audit log berhasil", func(t *testing.T) {
		var passwordResetOK bool
		repo := &repositories.MockAdminUserRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*models.AdminUser, error) {
				return &models.AdminUser{
					ID:       "staff-petugas-1",
					Username: "petugas1",
					Name:     "Petugas 1",
					Role:     "PETUGAS",
					IsActive: true,
				}, nil
			},
			ResetStaffPasswordFunc: func(ctx context.Context, id string, passwordHash string) error {
				if utils.CheckPasswordHash("RahasiaBaru99", passwordHash) {
					passwordResetOK = true
				}
				return nil
			},
			GetAuditLogsFunc: func(ctx context.Context, actionFilter, search string, limit int) ([]models.AuditLog, error) {
				return []models.AuditLog{{
					ID:        "audit-1",
					Action:    "RESET_STAFF_PASSWORD",
					ActorRole: "SUPER_ADMIN",
				}}, nil
			},
		}
		app := setupStaffControllerTestApp(repo)

		reqReset := httptest.NewRequest(http.MethodPost, "/api/v1/admin/staff/staff-petugas-1/reset-password", bytes.NewReader([]byte(`{"newPassword":"RahasiaBaru99"}`)))
		reqReset.Header.Set("Authorization", "Bearer "+superToken)
		reqReset.Header.Set("Content-Type", "application/json")
		respReset, err := app.Test(reqReset, -1)
		if err != nil || respReset.StatusCode != http.StatusOK || !passwordResetOK {
			t.Fatalf("Reset password gagal: err=%v status=%v ok=%v", err, respReset, passwordResetOK)
		}

		reqLogs := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
		reqLogs.Header.Set("Authorization", "Bearer "+superToken)
		respLogs, err := app.Test(reqLogs, -1)
		if err != nil || respLogs.StatusCode != http.StatusOK {
			t.Fatalf("GetAuditLogs gagal: err=%v status=%v", err, respLogs)
		}
	})
}

func TestAuditTrailMiddleware(t *testing.T) {
	t.Setenv("JWT_SECRET", "symphoniatic-test-jwt-secret-key-min-32")

	adminToken, err := utils.GenerateUserToken("admin-1", "admin@symphoniatic.id", "ADMIN")
	if err != nil {
		t.Fatalf("Gagal membuat token ADMIN: %v", err)
	}

	var recordedActions []string
	repo := &repositories.MockAdminUserRepository{
		CreateAuditLogFunc: func(ctx context.Context, logEntry *models.AuditLog) error {
			recordedActions = append(recordedActions, logEntry.Action)
			return nil
		},
	}

	app := fiber.New()
	admin := app.Group("/api/v1/admin")
	admin.Use(controllers.AuditTrailMiddleware(repo))

	guard := middleware.RequireAdminAuth("ADMIN")
	admin.Get("/events", guard, func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	admin.Post("/events", guard, func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusCreated)
	})
	admin.Patch("/refunds/:id/status", guard, func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	admin.Delete("/promos/:id", guard, func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusBadRequest)
	})

	// GET tidak boleh dicatat
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/admin/events", nil)
	reqGet.Header.Set("Authorization", "Bearer "+adminToken)
	_, _ = app.Test(reqGet, -1)

	// POST sukses (201) -> dicatat sebagai CREATE_EVENT
	reqPost := httptest.NewRequest(http.MethodPost, "/api/v1/admin/events", nil)
	reqPost.Header.Set("Authorization", "Bearer "+adminToken)
	_, _ = app.Test(reqPost, -1)

	// PATCH sukses (200) -> dicatat sebagai UPDATE_REFUND_STATUS
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/refunds/ref-1/status", nil)
	reqPatch.Header.Set("Authorization", "Bearer "+adminToken)
	_, _ = app.Test(reqPatch, -1)

	// DELETE gagal (400) -> tidak boleh dicatat
	reqFail := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/promos/promo-1", nil)
	reqFail.Header.Set("Authorization", "Bearer "+adminToken)
	_, _ = app.Test(reqFail, -1)

	if len(recordedActions) != 2 {
		t.Fatalf("Ekspektasi 2 audit log tercatat, dapat %d: %v", len(recordedActions), recordedActions)
	}
	if recordedActions[0] != "CREATE_EVENT" || recordedActions[1] != "UPDATE_REFUND_STATUS" {
		t.Fatalf("Urutan aksi audit log tidak sesuai: %v", recordedActions)
	}
}
