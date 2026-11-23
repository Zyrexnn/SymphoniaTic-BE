package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

func setupSecurityTestApp(secRepo repositories.SecurityRepository, auditRepo repositories.AdminUserRepository, role string) (*fiber.App, *middleware.IPBlocklistManager) {
	blockMgr := middleware.NewIPBlocklistManager(secRepo, auditRepo)
	ctrl := NewSecurityController(secRepo, auditRepo, blockMgr)

	app := fiber.New()
	admin := app.Group("/api/v1/admin", func(ctx *fiber.Ctx) error {
		ctx.Locals("admin_id", "staff-1")
		ctx.Locals("admin_email", "superadmin@symphoniatic.id")
		ctx.Locals("admin_role", role)
		return ctx.Next()
	})
	guard := func(ctx *fiber.Ctx) error {
		if ctx.Locals("admin_role") != "SUPER_ADMIN" {
			return ctx.Status(fiber.StatusForbidden).JSON(models.APIResponse{Success: false, Message: "Forbidden"})
		}
		return ctx.Next()
	}

	admin.Get("/audit-logs/search", guard, ctrl.GetFilteredAuditLogs)
	admin.Get("/security/blocked-ips", guard, ctrl.ListBlockedIPs)
	admin.Post("/security/blocked-ips", guard, ctrl.BlockIP)
	admin.Delete("/security/blocked-ips/:id", guard, ctrl.UnblockIP)
	return app, blockMgr
}

func TestGetFilteredAuditLogs_ForwardsQueryParams(t *testing.T) {
	var captured models.AuditLogFilter
	secRepo := &repositories.MockSecurityRepository{
		GetAuditLogsFilteredFunc: func(ctx context.Context, filter models.AuditLogFilter) (*models.AuditLogPage, error) {
			captured = filter
			return &models.AuditLogPage{
				Items: []models.AuditLog{{ID: "a-1", Category: "SECURITY", Severity: "CRITICAL", Action: "RATE_LIMIT_EXCEEDED"}},
				Total: 1,
				Page:  filter.Page,
				Limit: filter.Limit,
			}, nil
		},
	}

	app, _ := setupSecurityTestApp(secRepo, &repositories.MockAdminUserRepository{}, "SUPER_ADMIN")
	req := httptest.NewRequest("GET", "/api/v1/admin/audit-logs/search?category=SECURITY&severity=CRITICAL&ip=10.0.0.1&from=2026-11-01&to=2026-11-23&page=2&limit=25", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Ekspektasi 200, dapat %d", resp.StatusCode)
	}
	if captured.Category != "SECURITY" || captured.Severity != "CRITICAL" || captured.IPAddress != "10.0.0.1" || captured.Page != 2 || captured.Limit != 25 {
		t.Fatalf("Filter tidak diteruskan dengan benar: %+v", captured)
	}
	if captured.From == nil || captured.To == nil {
		t.Fatalf("Tanggal from/to gagal diparsing: from=%v to=%v", captured.From, captured.To)
	}
}

func TestBlockIP_RejectsInvalidAndSelfIP(t *testing.T) {
	app, _ := setupSecurityTestApp(&repositories.MockSecurityRepository{}, &repositories.MockAdminUserRepository{}, "SUPER_ADMIN")
	t.Setenv("APP_ENV", "development")

	badBody, _ := json.Marshal(models.BlockIPRequest{IPAddress: "bukan-ip", Reason: "test"})
	reqBad := httptest.NewRequest("POST", "/api/v1/admin/security/blocked-ips", bytes.NewReader(badBody))
	reqBad.Header.Set("Content-Type", "application/json")
	respBad, _ := app.Test(reqBad)
	if respBad.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Ekspektasi 400 untuk IP invalid, dapat %d", respBad.StatusCode)
	}

	selfBody, _ := json.Marshal(models.BlockIPRequest{IPAddress: "198.51.100.1", Reason: "self test"})
	reqSelf := httptest.NewRequest("POST", "/api/v1/admin/security/blocked-ips", bytes.NewReader(selfBody))
	reqSelf.Header.Set("Content-Type", "application/json")
	reqSelf.Header.Set("X-Test-Client-IP", "198.51.100.1")
	respSelf, _ := app.Test(reqSelf)
	if respSelf.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("Ekspektasi 400 saat memblokir IP sendiri, dapat %d", respSelf.StatusCode)
	}
}

func TestBlockAndUnblockIP_SuccessAndAudited(t *testing.T) {
	var actions []string
	auditRepo := &repositories.MockAdminUserRepository{
		CreateAuditLogFunc: func(ctx context.Context, logEntry *models.AuditLog) error {
			actions = append(actions, logEntry.Action)
			return nil
		},
	}
	secRepo := &repositories.MockSecurityRepository{
		BlockIPFunc: func(ctx context.Context, entry *models.BlockedIP) (*models.BlockedIP, error) {
			entry.ID = "blk-123"
			return entry, nil
		},
	}

	app, _ := setupSecurityTestApp(secRepo, auditRepo, "SUPER_ADMIN")
	body, _ := json.Marshal(models.BlockIPRequest{IPAddress: "203.0.113.77", Reason: "Brute force", DurationMinutes: 60})
	reqBlock := httptest.NewRequest("POST", "/api/v1/admin/security/blocked-ips", bytes.NewReader(body))
	reqBlock.Header.Set("Content-Type", "application/json")
	respBlock, _ := app.Test(reqBlock)
	if respBlock.StatusCode != fiber.StatusCreated {
		t.Fatalf("Ekspektasi 201 saat block IP, dapat %d", respBlock.StatusCode)
	}

	reqUnblock := httptest.NewRequest("DELETE", "/api/v1/admin/security/blocked-ips/203.0.113.77", nil)
	respUnblock, _ := app.Test(reqUnblock)
	if respUnblock.StatusCode != fiber.StatusOK {
		t.Fatalf("Ekspektasi 200 saat unblock IP, dapat %d", respUnblock.StatusCode)
	}

	if len(actions) != 2 || actions[0] != "BLOCK_IP" || actions[1] != "UNBLOCK_IP" {
		t.Fatalf("Ekspektasi audit log BLOCK_IP & UNBLOCK_IP, dapat %v", actions)
	}
}

func TestSecurityRoutes_RequireSuperAdmin(t *testing.T) {
	app, _ := setupSecurityTestApp(&repositories.MockSecurityRepository{}, &repositories.MockAdminUserRepository{}, "ADMIN")
	req := httptest.NewRequest("GET", "/api/v1/admin/security/blocked-ips", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("Ekspektasi 403 untuk role ADMIN, dapat %d", resp.StatusCode)
	}
}
