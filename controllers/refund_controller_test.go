package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	userGroup := api.Group("/user", middleware.RequireUserAuth)
	userGroup.Post("/orders/:orderCode/refund", ctrl.SubmitUserRefund)

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

	t.Run("Gagal (400 Bad Request) saat pengajuan refund kurang dari 24 jam sebelum konser", func(t *testing.T) {
		soonConcert := time.Now().Add(5 * time.Hour)
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					OrderCode: "SYM-123456",
					UserEmail: "user@example.com",
					Status:    "ISSUED",
					Date:      soonConcert.Format("2006-01-02"),
					EventTime: soonConcert.Format("15:04 WIB"),
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
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request untuk H-5 jam, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (429 Too Many Requests) saat permohonan OTP masih dalam masa cooldown 60 detik", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					OrderCode: "SYM-123456",
					UserEmail: "user@example.com",
					Status:    "ISSUED",
				}, nil
			},
		}
		mockRefundRepo := &repositories.MockRefundRepository{
			RequestRefundOTPFunc: func(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error {
				return repositories.ErrOTPCooldown
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

		if resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("Status tidak sesuai: ekspektasi 429 Too Many Requests, dapat %d", resp.StatusCode)
		}

		var apiResp models.APIResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiResp)
		if apiResp.Message != "Mohon tunggu 60 detik sebelum meminta kode OTP refund baru" {
			t.Fatalf("Pesan respons tidak sesuai: dapat %s", apiResp.Message)
		}
	})

	t.Run("Sukses (200 OK) saat status tiket REMINDED", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:         "ord-reminded-01",
					OrderCode:  "SYM-123456",
					UserEmail:  "user@example.com",
					TotalPrice: 750000,
					Status:     "REMINDED",
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
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK untuk tiket REMINDED, dapat %d", resp.StatusCode)
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

	t.Run("Gagal (400 Bad Request) saat batas percobaan verifikasi OTP terlampaui", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			SubmitRefundFunc: func(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error) {
				return nil, repositories.ErrOTPMaxExceed
			},
		}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.SubmitRefundInput{
			OrderCode:     "SYM-123456",
			UserEmail:     "user@example.com",
			OTPCode:       "111111",
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

		var apiResp models.APIResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiResp)
		if apiResp.Message != "Batas percobaan verifikasi OTP telah terlampaui. Silakan minta kode OTP baru" {
			t.Fatalf("Pesan respons tidak sesuai: dapat %s", apiResp.Message)
		}
	})

	t.Run("Gagal (400 Bad Request) saat kode OTP salah dan memberitahu sisa percobaan", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			SubmitRefundFunc: func(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error) {
				return nil, &repositories.ErrOTPWrongCode{RemainingAttempts: 3}
			},
		}
		mockOrderRepo := &repositories.MockOrderRepository{}

		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		payload := models.SubmitRefundInput{
			OrderCode:     "SYM-123456",
			UserEmail:     "user@example.com",
			OTPCode:       "999999",
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

		var apiResp models.APIResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiResp)
		if apiResp.Message != "Kode OTP salah. Sisa percobaan: 3" {
			t.Fatalf("Pesan respons tidak sesuai: dapat %s", apiResp.Message)
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

func TestRefundController_SubmitUserRefund(t *testing.T) {
	t.Setenv("JWT_SECRET", "symphoniatic-test-jwt-secret-key-min-32")

	memberToken, err := utils.GenerateUserToken("usr-101", "member@example.com", "USER")
	if err != nil {
		t.Fatalf("Gagal membuat token member: %v", err)
	}

	t.Run("Sukses (201 Created) pengajuan refund 1-klik untuk member pemilik pesanan", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:         "ord-101",
					OrderCode:  "SYM-MEM101",
					UserID:     "usr-101",
					UserEmail:  "member@example.com",
					TotalPrice: 450000,
					Status:     "ISSUED",
				}, nil
			},
		}
		mockRefundRepo := &repositories.MockRefundRepository{
			SubmitDirectUserRefundFunc: func(ctx context.Context, order *models.OrderRecord, input *models.SubmitUserRefundInput) (*models.RefundRequestRecord, error) {
				return &models.RefundRequestRecord{
					ID:           "rf-direct-1",
					OrderID:      order.ID,
					OrderCode:    order.OrderCode,
					UserEmail:    order.UserEmail,
					BankName:     input.BankName,
					RefundAmount: order.TotalPrice,
					Status:       "PENDING",
				}, nil
			},
		}
		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)

		payload := models.SubmitUserRefundInput{
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountHolder: "Anggota Simfoni",
			Reason:        "Berhalangan hadir karena tugas luar kota",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-MEM101/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Ekspektasi 201 Created, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (403 Forbidden) jika pesanan milik akun lain", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:        "ord-999",
					OrderCode: "SYM-OTHER",
					UserID:    "usr-other",
					UserEmail: "other@example.com",
					Status:    "ISSUED",
				}, nil
			},
		}
		app := setupRefundControllerTest(&repositories.MockRefundRepository{}, mockOrderRepo)

		payload := models.SubmitUserRefundInput{
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountHolder: "Penyerobot",
			Reason:        "Tes",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-OTHER/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Ekspektasi 403 Forbidden, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) jika tiket sudah check-in", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:        "ord-checked",
					OrderCode: "SYM-CHK01",
					UserID:    "usr-101",
					UserEmail: "member@example.com",
					Status:    "CHECKED_IN",
				}, nil
			},
		}
		app := setupRefundControllerTest(&repositories.MockRefundRepository{}, mockOrderRepo)

		payload := models.SubmitUserRefundInput{
			BankName:      "Mandiri",
			AccountNumber: "999888777",
			AccountHolder: "Anggota",
			Reason:        "Tes",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-CHK01/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) jika kolom rekening bank kosong", func(t *testing.T) {
		app := setupRefundControllerTest(&repositories.MockRefundRepository{}, &repositories.MockOrderRepository{})

		payload := models.SubmitUserRefundInput{BankName: "BCA"}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-MEM101/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) jika pengajuan refund sudah pernah diajukan", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:        "ord-101",
					OrderCode: "SYM-MEM101",
					UserID:    "usr-101",
					UserEmail: "member@example.com",
					Status:    "ISSUED",
				}, nil
			},
		}
		mockRefundRepo := &repositories.MockRefundRepository{
			SubmitDirectUserRefundFunc: func(ctx context.Context, order *models.OrderRecord, input *models.SubmitUserRefundInput) (*models.RefundRequestRecord, error) {
				return nil, repositories.ErrRefundAlreadyExists
			},
		}
		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)

		payload := models.SubmitUserRefundInput{
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountHolder: "Anggota Simfoni",
			Reason:        "Jadwal bentrok",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-MEM101/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})
}

func TestRefundController_EventDisruptionBypass(t *testing.T) {
	memberToken, _ := utils.GenerateUserToken("usr-101", "member@example.com", "Anggota Simfoni")
	nearConcertDate := time.Now().Add(2 * time.Hour).Format("2006-01-02")
	nearConcertTime := time.Now().Add(2 * time.Hour).Format("15:04")

	t.Run("Bypass batas 24 jam berhasil (201 Created) saat event CANCELLED", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:          "ord-cancel-1",
					OrderCode:   "SYM-CANCEL1",
					UserID:      "usr-101",
					UserEmail:   "member@example.com",
					EventStatus: "CANCELLED",
					Date:        nearConcertDate,
					EventTime:   nearConcertTime,
					Status:      "ISSUED",
					TotalPrice:  500000,
				}, nil
			},
		}
		mockRefundRepo := &repositories.MockRefundRepository{
			SubmitDirectUserRefundFunc: func(ctx context.Context, order *models.OrderRecord, input *models.SubmitUserRefundInput) (*models.RefundRequestRecord, error) {
				return &models.RefundRequestRecord{ID: "ref-1", OrderCode: order.OrderCode, Status: "PENDING"}, nil
			},
		}
		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		bodyBytes, _ := json.Marshal(models.SubmitUserRefundInput{
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountHolder: "Anggota Simfoni",
			Reason:        "Konser dibatalkan penyelenggara",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-CANCEL1/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Ekspektasi 201 Created saat event CANCELLED, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Bypass batas 24 jam berhasil (201 Created) saat event RESCHEDULED", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:          "ord-resched-1",
					OrderCode:   "SYM-RESCH1",
					UserID:      "usr-101",
					UserEmail:   "member@example.com",
					EventStatus: "RESCHEDULED",
					Date:        nearConcertDate,
					EventTime:   nearConcertTime,
					Status:      "ISSUED",
					TotalPrice:  500000,
				}, nil
			},
		}
		mockRefundRepo := &repositories.MockRefundRepository{
			SubmitDirectUserRefundFunc: func(ctx context.Context, order *models.OrderRecord, input *models.SubmitUserRefundInput) (*models.RefundRequestRecord, error) {
				return &models.RefundRequestRecord{ID: "ref-2", OrderCode: order.OrderCode, Status: "PENDING"}, nil
			},
		}
		app := setupRefundControllerTest(mockRefundRepo, mockOrderRepo)
		bodyBytes, _ := json.Marshal(models.SubmitUserRefundInput{
			BankName:      "Mandiri",
			AccountNumber: "9876543210",
			AccountHolder: "Anggota Simfoni",
			Reason:        "Tidak bisa hadir di jadwal baru",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-RESCH1/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Ekspektasi 201 Created saat event RESCHEDULED, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Tetap ditolak (400 Bad Request) saat event CANCELLED tetapi tiket sudah CHECKED_IN", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:          "ord-used-1",
					OrderCode:   "SYM-USED1",
					UserID:      "usr-101",
					UserEmail:   "member@example.com",
					EventStatus: "CANCELLED",
					Status:      "CHECKED_IN",
				}, nil
			},
		}
		app := setupRefundControllerTest(&repositories.MockRefundRepository{}, mockOrderRepo)
		bodyBytes, _ := json.Marshal(models.SubmitUserRefundInput{
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountHolder: "Anggota Simfoni",
			Reason:        "Konser dibatalkan",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-USED1/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request untuk tiket CHECKED_IN, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Tetap ditolak (400 Bad Request) saat event SCHEDULED dan konser < 24 jam", func(t *testing.T) {
		mockOrderRepo := &repositories.MockOrderRepository{
			GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:          "ord-sched-1",
					OrderCode:   "SYM-SCHED1",
					UserID:      "usr-101",
					UserEmail:   "member@example.com",
					EventStatus: "SCHEDULED",
					Date:        nearConcertDate,
					EventTime:   nearConcertTime,
					Status:      "ISSUED",
				}, nil
			},
		}
		app := setupRefundControllerTest(&repositories.MockRefundRepository{}, mockOrderRepo)
		bodyBytes, _ := json.Marshal(models.SubmitUserRefundInput{
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountHolder: "Anggota Simfoni",
			Reason:        "Berubah pikiran",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-SCHED1/refund", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request untuk event SCHEDULED < 24 jam, dapat %d", resp.StatusCode)
		}
	})
}

func TestRefundController_AdminBatchRefundByEvent(t *testing.T) {
	t.Run("Sukses (200 OK) memproses batch refund konser batal", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			BatchRefundByEventIDFunc: func(ctx context.Context, eventID string) (*models.BatchRefundResult, []models.OrderRecord, error) {
				return &models.BatchRefundResult{
						EventID:           eventID,
						ProcessedCount:    2,
						SkippedCount:      1,
						TotalRefundAmount: 1500000,
					}, []models.OrderRecord{
						{OrderCode: "SYM-1", UserEmail: "a@example.com", TotalPrice: 750000},
						{OrderCode: "SYM-2", UserEmail: "b@example.com", TotalPrice: 750000},
					}, nil
			},
		}
		app := fiber.New()
		ctrl := controllers.NewRefundControllerWithOrderRepo(mockRefundRepo, &repositories.MockOrderRepository{})
		app.Post("/api/v1/admin/events/:id/batch-refund", ctrl.AdminBatchRefundByEvent)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/events/evt-1/batch-refund", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) jika konser belum dibatalkan", func(t *testing.T) {
		mockRefundRepo := &repositories.MockRefundRepository{
			BatchRefundByEventIDFunc: func(ctx context.Context, eventID string) (*models.BatchRefundResult, []models.OrderRecord, error) {
				return nil, nil, errors.New("batch refund hanya dapat dijalankan untuk konser yang sudah dibatalkan")
			},
		}
		app := fiber.New()
		ctrl := controllers.NewRefundControllerWithOrderRepo(mockRefundRepo, &repositories.MockOrderRepository{})
		app.Post("/api/v1/admin/events/:id/batch-refund", ctrl.AdminBatchRefundByEvent)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/events/evt-1/batch-refund", nil)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})
}
