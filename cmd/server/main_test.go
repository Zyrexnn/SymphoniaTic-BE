package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
)

func setupTestRouter() *config.Config {
	return &config.Config{
		Auth: config.AuthConfig{
			AdminUsername:   "admin",
			AdminPassword:   "admin123",
			PetugasUsername: "petugas",
			PetugasPassword: "petugas123",
			JWTSecret:       "symphoniatic-test-jwt-secret-key-min-32",
		},
	}
}

func TestRouter_RoleAuthorizationSegregation(t *testing.T) {
	t.Setenv("JWT_SECRET", "symphoniatic-test-jwt-secret-key-min-32")

	cfg := setupTestRouter()
	eventRepo := &repositories.MockEventRepository{}
	orderRepo := &repositories.MockOrderRepository{
		GetAllOrdersFunc: func(ctx context.Context, searchQuery, statusFilter string) ([]models.OrderRecord, error) {
			return []models.OrderRecord{}, nil
		},
		GetAdminDashboardMetricsFunc: func(ctx context.Context) (*models.AdminDashboardMetrics, error) {
			return &models.AdminDashboardMetrics{}, nil
		},
		CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
			return &models.OrderRecord{OrderCode: orderCode, Status: "CHECKED_IN"}, nil
		},
		GetAttendanceSummaryFunc: func(ctx context.Context, eventID string) (*models.GateAttendanceSummary, error) {
			return &models.GateAttendanceSummary{TotalTicketsIssued: 10, TotalCheckedIn: 4, RemainingCheckIn: 6, CheckInPercentage: 40.0}, nil
		},
	}
	refundRepo := &repositories.MockRefundRepository{
		GetAllRefundsFunc: func(ctx context.Context) ([]models.RefundRequestRecord, error) {
			return []models.RefundRequestRecord{}, nil
		},
		UpdateRefundStatusFunc: func(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error) {
			return &models.RefundRequestRecord{ID: refundID, Status: newStatus}, nil
		},
	}

	app := setupRouter(cfg, eventRepo, orderRepo, refundRepo)

	petugasToken, err := utils.GenerateUserToken("petugas-1", "petugas@symphoniatic.id", "PETUGAS")
	if err != nil {
		t.Fatalf("Gagal menerbitkan token PETUGAS: %v", err)
	}

	adminToken, err := utils.GenerateUserToken("admin-1", "admin@symphoniatic.id", "ADMIN")
	if err != nil {
		t.Fatalf("Gagal menerbitkan token ADMIN: %v", err)
	}

	adminExclusiveEndpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/admin/dashboard", ""},
		{http.MethodPost, "/api/v1/admin/upload", ""},
		{http.MethodPost, "/api/v1/admin/events", "{}"},
		{http.MethodPut, "/api/v1/admin/events/evt-1", "{}"},
		{http.MethodPatch, "/api/v1/admin/events/evt-1/toggle-close", ""},
		{http.MethodDelete, "/api/v1/admin/events/evt-1", ""},
		{http.MethodPost, "/api/v1/admin/events/evt-1/categories", "{}"},
		{http.MethodPut, "/api/v1/admin/categories/cat-1", "{}"},
		{http.MethodDelete, "/api/v1/admin/categories/cat-1", ""},
		{http.MethodPatch, "/api/v1/admin/orders/ord-1/status", `{"status":"CANCELLED"}`},
		{http.MethodGet, "/api/v1/admin/refunds", ""},
		{http.MethodPatch, "/api/v1/admin/refunds/ref-1/status", `{"status":"APPROVED"}`},
	}

	for _, tc := range adminExclusiveEndpoints {
		t.Run("PETUGAS ditolak 403 pada "+tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+petugasToken)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Gagal eksekusi request: %v", err)
			}
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("Ekspektasi 403 Forbidden untuk PETUGAS pada %s %s, dapat: %d", tc.method, tc.path, resp.StatusCode)
			}
		})

		t.Run("ADMIN lolos otorisasi pada "+tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Gagal eksekusi request: %v", err)
			}
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				t.Errorf("ADMIN tidak boleh ditolak (401/403) pada %s %s, dapat: %d", tc.method, tc.path, resp.StatusCode)
			}
		})
	}

	gateSharedEndpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/admin/orders", ""},
		{http.MethodPost, "/api/v1/admin/tickets/checkin", `{"code":"SYM-123456"}`},
		{http.MethodGet, "/api/v1/admin/tickets/attendance-summary", ""},
	}

	for _, tc := range gateSharedEndpoints {
		t.Run("PETUGAS diizinkan pada "+tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+petugasToken)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Gagal eksekusi request: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Ekspektasi 200 OK untuk PETUGAS pada %s %s, dapat: %d", tc.method, tc.path, resp.StatusCode)
			}
		})
	}
}

func TestRouter_RateLimiting(t *testing.T) {
	t.Setenv("JWT_SECRET", "symphoniatic-test-jwt-secret-key-min-32")

	cfg := setupTestRouter()
	eventRepo := &repositories.MockEventRepository{}
	orderRepo := &repositories.MockOrderRepository{
		LookupTicketByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
			return nil, repositories.ErrNotFound
		},
	}
	refundRepo := &repositories.MockRefundRepository{}
	app := setupRouter(cfg, eventRepo, orderRepo, refundRepo)

	t.Run("Menolak permintaan berlebih pada /tickets/lookup dengan 429 Too Many Requests", func(t *testing.T) {
		var lastStatus int
		for i := 0; i < 31; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/lookup?code=SYM-000001", nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Gagal mengirim request ke-%d: %v", i, err)
			}
			lastStatus = resp.StatusCode
		}
		if lastStatus != http.StatusTooManyRequests {
			t.Fatalf("Ekspektasi 429 Too Many Requests pada request ke-31, dapat: %d", lastStatus)
		}
	})

	t.Run("Menolak login admin berulang dengan 429 Too Many Requests", func(t *testing.T) {
		var lastStatus int
		for i := 0; i < 11; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Gagal mengirim request login ke-%d: %v", i, err)
			}
			lastStatus = resp.StatusCode
		}
		if lastStatus != http.StatusTooManyRequests {
			t.Fatalf("Ekspektasi 429 Too Many Requests pada request login ke-11, dapat: %d", lastStatus)
		}
	})
}
