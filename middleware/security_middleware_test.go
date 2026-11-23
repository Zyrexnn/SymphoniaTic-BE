package middleware

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

func TestBlocklistMiddleware_RejectsActiveAndAllowsExpired(t *testing.T) {
	mgr := NewIPBlocklistManager(&repositories.MockSecurityRepository{}, &repositories.MockAdminUserRepository{})

	future := time.Now().Add(10 * time.Minute)
	_, err := mgr.BlockIP(context.Background(), &models.BlockedIP{
		IPAddress: "203.0.113.10",
		Reason:    "Uji blokir aktif",
		BlockedBy: "superadmin",
		ExpiresAt: &future,
	})
	if err != nil {
		t.Fatalf("BlockIP gagal: %v", err)
	}

	past := time.Now().Add(-1 * time.Minute)
	_, err = mgr.BlockIP(context.Background(), &models.BlockedIP{
		IPAddress: "203.0.113.11",
		Reason:    "Uji blokir kedaluwarsa",
		BlockedBy: "superadmin",
		ExpiresAt: &past,
	})
	if err != nil {
		t.Fatalf("BlockIP gagal: %v", err)
	}

	t.Setenv("APP_ENV", "development")

	app := fiber.New()
	app.Use(mgr.BlocklistMiddleware())
	app.Get("/ping", func(ctx *fiber.Ctx) error {
		return ctx.SendString("ok")
	})

	reqBlocked := httptest.NewRequest("GET", "/ping", nil)
	reqBlocked.Header.Set("X-Test-Client-IP", "203.0.113.10")
	respBlocked, err := app.Test(reqBlocked)
	if err != nil {
		t.Fatalf("app.Test gagal: %v", err)
	}
	if respBlocked.StatusCode != fiber.StatusForbidden {
		t.Fatalf("Ekspektasi 403 untuk IP diblokir, dapat %d", respBlocked.StatusCode)
	}

	reqExpired := httptest.NewRequest("GET", "/ping", nil)
	reqExpired.Header.Set("X-Test-Client-IP", "203.0.113.11")
	respExpired, err := app.Test(reqExpired)
	if err != nil {
		t.Fatalf("app.Test gagal: %v", err)
	}
	if respExpired.StatusCode != fiber.StatusOK {
		t.Fatalf("Ekspektasi 200 untuk IP kedaluwarsa, dapat %d", respExpired.StatusCode)
	}
}

func TestResolveClientIP_HeaderOnlyInDevelopment(t *testing.T) {
	mgr := NewIPBlocklistManager(nil, nil)
	app := fiber.New()
	var resolved string
	app.Get("/ip", func(ctx *fiber.Ctx) error {
		resolved = mgr.ResolveClientIP(ctx)
		return ctx.SendString(resolved)
	})

	t.Setenv("APP_ENV", "production")
	reqProd := httptest.NewRequest("GET", "/ip", nil)
	reqProd.Header.Set("X-Test-Client-IP", "198.51.100.7")
	_, _ = app.Test(reqProd)
	if resolved == "198.51.100.7" {
		t.Fatalf("Header X-Test-Client-IP tidak boleh dipakai di production")
	}

	t.Setenv("APP_ENV", "development")
	reqDev := httptest.NewRequest("GET", "/ip", nil)
	reqDev.Header.Set("X-Test-Client-IP", "198.51.100.7")
	_, _ = app.Test(reqDev)
	if resolved != "198.51.100.7" {
		t.Fatalf("Ekspektasi 198.51.100.7 di development, dapat %s", resolved)
	}
}

func TestSecurityAuditMiddleware_LogsAndTriggersAutoBan(t *testing.T) {
	var mu sync.Mutex
	var recordedLogs []models.AuditLog
	var blockedEntries []models.BlockedIP

	auditMock := &repositories.MockAdminUserRepository{
		CreateAuditLogFunc: func(ctx context.Context, logEntry *models.AuditLog) error {
			mu.Lock()
			recordedLogs = append(recordedLogs, *logEntry)
			mu.Unlock()
			return nil
		},
	}
	secMock := &repositories.MockSecurityRepository{
		BlockIPFunc: func(ctx context.Context, entry *models.BlockedIP) (*models.BlockedIP, error) {
			mu.Lock()
			blockedEntries = append(blockedEntries, *entry)
			mu.Unlock()
			return entry, nil
		},
	}

	mgr := NewIPBlocklistManager(secMock, auditMock)
	mgr.AutoBanThreshold = 10
	t.Setenv("APP_ENV", "development")

	app := fiber.New()
	app.Use(mgr.BlocklistMiddleware())
	app.Use(mgr.SecurityAuditMiddleware())
	app.Post("/api/v1/auth/login/password", func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusUnauthorized)
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login/password", nil)
		req.Header.Set("X-Test-Client-IP", "192.0.2.55")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test gagal pada iterasi %d: %v", i, err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("Ekspektasi 401 pada percobaan %d, dapat %d", i+1, resp.StatusCode)
		}
	}

	reqBanned := httptest.NewRequest("POST", "/api/v1/auth/login/password", nil)
	reqBanned.Header.Set("X-Test-Client-IP", "192.0.2.55")
	respBanned, err := app.Test(reqBanned)
	if err != nil {
		t.Fatalf("app.Test ke-11 gagal: %v", err)
	}
	if respBanned.StatusCode != fiber.StatusForbidden {
		t.Fatalf("Ekspektasi 403 setelah auto-ban, dapat %d", respBanned.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(blockedEntries) != 1 || !blockedEntries[0].IsAuto {
		t.Fatalf("Ekspektasi 1 auto-ban entry, dapat %+v", blockedEntries)
	}
	if len(recordedLogs) < 11 {
		t.Fatalf("Ekspektasi minimal 11 audit log (10 AUTH + 1 AUTO_BLOCK_IP), dapat %d", len(recordedLogs))
	}
}

func TestIPBlocklistManager_ConcurrentAccess(t *testing.T) {
	mgr := NewIPBlocklistManager(&repositories.MockSecurityRepository{}, &repositories.MockAdminUserRepository{})
	t.Setenv("APP_ENV", "development")

	app := fiber.New()
	app.Use(mgr.BlocklistMiddleware())
	app.Use(mgr.SecurityAuditMiddleware())
	app.Get("/test", func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusUnauthorized)
	})

	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Test-Client-IP", "198.51.100.88")
			_, _ = app.Test(req)
		}(g)
	}
	wg.Wait()
}
