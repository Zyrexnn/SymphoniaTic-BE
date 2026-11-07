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
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

func setupRefundControllerTest(mockRefundRepo *repositories.MockRefundRepository, mockOrderRepo *repositories.MockOrderRepository) *fiber.App {
	app := fiber.New()
	ctrl := controllers.NewRefundControllerWithOrderRepo(mockRefundRepo, mockOrderRepo)

	api := app.Group("/api/v1")
	api.Post("/refunds/request-otp", ctrl.RequestRefundOTP)
	api.Post("/refunds/submit", ctrl.SubmitRefund)
	api.Post("/refunds/status", ctrl.GetRefundStatus)

	admin := api.Group("/admin/refunds")
	admin.Get("", ctrl.AdminGetAllRefunds)
	admin.Patch("/:id/status", ctrl.AdminUpdateRefundStatus)

	return app
}

func TestRefundController_RequestRefundOTP(t *testing.T) {
	t.Run("Sukses (200 OK) saat order valid dan email cocok", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:         "ord-ref-01",
					OrderCode:  "SYM-123456",
					UserEmail:  "user@example.com",
					TotalPrice: 750000,
					Status:     "ISSUED",
				}, nil
			},
		}

		mockRefundRepo := &repositories.MockRefundRepository{
			RequestRefundOTPFunc: func(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error {
				return nil
			},
		}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.RequestRefundOTPInput{
			OrderCode: "SYM-123456",
			UserEmail: "user@example.com",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/request-otp", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}

		var apiResp models.APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			t.Fatalf("Gagal mendecode response body: %v", err)
		}
		if !apiResp.Success {
			t.Fatalf("Ekspektasi success bernilai true")
		}
	})

	t.Run("Gagal (400 Bad Request) saat order code atau email kosong", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{}
		mockOrderRepo := &repositories.MockOrderRepository{}
		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)

		payload := models.RequestRefundOTPInput{
			OrderCode: "",
			UserEmail: "",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/request-otp", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (404 Not Found) saat order code tidak ditemukan", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return nil, repositories.ErrNotFound
			},
		}
		mockRefundRepo := &repositories.MockRefundRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.RequestRefundOTPInput{
			OrderCode: "SYM-NOTFOUND",
			UserEmail: "user@example.com",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/request-otp", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (401 Unauthorized) saat email tidak cocok dengan pesanan", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					OrderCode: "SYM-123456",
					UserEmail: "owner@example.com",
					Status:    "ISSUED",
				}, nil
			},
		}
		mockRefundRepo := &repositories.MockRefundRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.RequestRefundOTPInput{
			OrderCode: "SYM-123456",
			UserEmail: "stranger@example.com",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/request-otp", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("Status tidak sesuai: ekspektasi 401 Unauthorized, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat tiket sudah CHECKED_IN", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					OrderCode: "SYM-123456",
					UserEmail: "user@example.com",
					Status:    "CHECKED_IN",
				}, nil
			},
		}
		mockRefundRepo := &repositories.MockRefundRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.RequestRefundOTPInput{
			OrderCode: "SYM-123456",
			UserEmail: "user@example.com",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/request-otp", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})
}

func TestRefundController_SubmitRefund(t *testing.T) {
	t.Run("Sukses (200 OK) saat OTP valid dan data lengkap", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			SubmitRefundFunc: func(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error) {
				return &models.RefundRequestRecord{
					ID:           "rf-001",
					OrderCode:    input.OrderCode,
					UserEmail:    input.UserEmail,
					RefundAmount: 500000,
					Status:       "PENDING",
				}, nil
			},
		}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.SubmitRefundInput{
			OrderCode:     "SYM-123456",
			UserEmail:     "user@example.com",
			OTPCode:       "888999",
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountHolder: "Budi Santoso",
			Reason:        "Ada tugas mendadak ke luar kota",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/submit", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}

		var apiResp models.APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			t.Fatalf("Gagal mendecode response body: %v", err)
		}
		if !apiResp.Success {
			t.Fatalf("Ekspektasi success bernilai true")
		}
	})

	t.Run("Gagal (400 Bad Request) saat OTP salah atau kedaluwarsa", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			SubmitRefundFunc: func(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error) {
				return nil, repositories.ErrOTPInvalid
			},
		}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.SubmitRefundInput{
			OrderCode:     "SYM-123456",
			UserEmail:     "user@example.com",
			OTPCode:       "000000",
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountHolder: "Budi Santoso",
			Reason:        "Alasan pembatalan",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/submit", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat kolom formulir tidak lengkap", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.SubmitRefundInput{
			OrderCode: "SYM-123456",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/submit", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})
}

func TestRefundController_GetRefundStatus(t *testing.T) {
	t.Run("Sukses (200 OK) saat permohonan refund ditemukan", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			GetRefundStatusFunc: func(ctx context.Context, orderCode, userEmail string) (*models.RefundRequestRecord, string, error) {
				return &models.RefundRequestRecord{
					ID:           "rf-001",
					OrderCode:    orderCode,
					UserEmail:    userEmail,
					Status:       "APPROVED",
					RefundAmount: 500000,
				}, "REFUND_REQUESTED", nil
			},
		}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.CheckRefundStatusRequest{
			OrderCode: "SYM-123456",
			UserEmail: "user@example.com",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/status", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}

		var apiResp models.APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			t.Fatalf("Gagal mendecode response body: %v", err)
		}
		if !apiResp.Success {
			t.Fatalf("Ekspektasi success bernilai true")
		}
	})

	t.Run("Gagal (404 Not Found) saat data refund tidak ditemukan", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			GetRefundStatusFunc: func(ctx context.Context, orderCode, userEmail string) (*models.RefundRequestRecord, string, error) {
				return nil, "", repositories.ErrNotFound
			},
		}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.CheckRefundStatusRequest{
			OrderCode: "SYM-NOTFOUND",
			UserEmail: "user@example.com",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/refunds/status", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})
}

func TestRefundController_AdminUpdateRefundStatus(t *testing.T) {
	t.Run("Sukses (200 OK) saat status diubah ke APPROVED", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			UpdateRefundStatusFunc: func(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error) {
				return &models.RefundRequestRecord{
					ID:           refundID,
					OrderCode:    "SYM-123456",
					UserEmail:    "user@example.com",
					RefundAmount: 500000,
					Status:       newStatus,
					AdminNote:    adminNote,
				}, nil
			},
		}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.UpdateRefundStatusRequest{
			Status:    "APPROVED",
			AdminNote: "Pengajuan disetujui, dana akan ditransfer dalam 2 hari kerja.",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/api/v1/admin/refunds/rf-001/status", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat status tidak valid", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.UpdateRefundStatusRequest{
			Status:    "INVALID_STATUS",
			AdminNote: "Catatan",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/api/v1/admin/refunds/rf-001/status", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (404 Not Found) saat ID refund tidak ditemukan", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			UpdateRefundStatusFunc: func(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error) {
				return nil, repositories.ErrNotFound
			},
		}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.UpdateRefundStatusRequest{
			Status:    "APPROVED",
			AdminNote: "Disetujui",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/api/v1/admin/refunds/rf-nonexistent/status", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})
}
