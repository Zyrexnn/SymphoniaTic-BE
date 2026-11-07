package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

func setupAdminControllerTest(mockRepo *repositories.MockOrderRepository, authCfg *config.AuthConfig) *fiber.App {
	app := fiber.New()
	adminCtrl := controllers.NewAdminController(mockRepo, authCfg)

	api := app.Group("/api/v1/admin")
	api.Post("/login", adminCtrl.AdminLogin)
	api.Get("/dashboard", adminCtrl.GetAdminDashboardMetrics)
	api.Post("/upload", adminCtrl.UploadImage)

	return app
}

func TestAdminController_AdminLogin(t *testing.T) {
	authCfg := &config.AuthConfig{
		AdminUsername:   "admin_symphonia",
		AdminPassword:   "AdminSecret2026!",
		PetugasUsername: "petugas_gate",
		PetugasPassword: "GateSecret2026!",
		JWTSecret:       "symphoniatic-jwt-secret-for-unit-tests-only-min-32-chars",
	}

	t.Run("Sukses dengan role ADMIN (200 OK & JWT token)", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupAdminControllerTest(mockRepo, authCfg)

		payload := models.AdminLoginRequest{
			Username: "admin_symphonia",
			Password: "AdminSecret2026!",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/login", bytes.NewBuffer(bodyBytes))
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
			t.Fatalf("Gagal mendecode response: %v", err)
		}

		dataMap, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Format payload data tidak valid")
		}

		token, ok := dataMap["token"].(string)
		if !ok || token == "" {
			t.Fatalf("JWT token tidak ditemukan pada response")
		}

		claims, err := utils.ValidateUserToken(token)
		if err != nil || claims.Role != "ADMIN" {
			t.Fatalf("Klaim role JWT tidak valid: %v, role: %s", err, claims.Role)
		}
	})

	t.Run("Sukses dengan role PETUGAS (200 OK & JWT token)", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupAdminControllerTest(mockRepo, authCfg)

		payload := models.AdminLoginRequest{
			Username: "petugas_gate",
			Password: "GateSecret2026!",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/login", bytes.NewBuffer(bodyBytes))
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
			t.Fatalf("Gagal mendecode response: %v", err)
		}

		dataMap, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Format payload data tidak valid")
		}

		token, ok := dataMap["token"].(string)
		if !ok || token == "" {
			t.Fatalf("JWT token tidak ditemukan pada response")
		}

		claims, err := utils.ValidateUserToken(token)
		if err != nil || claims.Role != "PETUGAS" {
			t.Fatalf("Klaim role JWT tidak valid: %v, role: %s", err, claims.Role)
		}
	})

	t.Run("Gagal dengan username/password salah (401 Unauthorized)", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupAdminControllerTest(mockRepo, authCfg)

		payload := models.AdminLoginRequest{
			Username: "admin_symphonia",
			Password: "WrongPassword!",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("Status tidak sesuai: ekspektasi 401 Unauthorized, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat payload JSON tidak valid", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupAdminControllerTest(mockRepo, authCfg)

		req := httptest.NewRequest("POST", "/api/v1/admin/login", bytes.NewBufferString("{invalid-json"))
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

func TestAdminController_GetAdminDashboardMetrics(t *testing.T) {
	t.Run("Sukses (200 OK) mengembalikan seluruh metrik finansial & penjualan", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			GetAdminDashboardMetricsFunc: func(ctx context.Context) (*models.AdminDashboardMetrics, error) {
				return &models.AdminDashboardMetrics{
					TotalRevenue:   150000000,
					TicketsSold:    450,
					RemainingQuota: 220,
					TotalEvents:    8,
					TotalOrders:    300,
					RecentOrders: []models.RecentOrderSummary{
						{
							ID:         "ord-recent-01",
							OrderCode:  "SYM-999001",
							TotalPrice: 750000,
							Status:     "ISSUED",
						},
					},
				}, nil
			},
		}

		app := setupAdminControllerTest(mockRepo, nil)
		req := httptest.NewRequest("GET", "/api/v1/admin/dashboard", nil)
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

	t.Run("Gagal (500 Internal Server Error) saat database error", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			GetAdminDashboardMetricsFunc: func(ctx context.Context) (*models.AdminDashboardMetrics, error) {
				return nil, fmt.Errorf("koneksi database terputus")
			},
		}

		app := setupAdminControllerTest(mockRepo, nil)
		req := httptest.NewRequest("GET", "/api/v1/admin/dashboard", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("Status tidak sesuai: ekspektasi 500 Internal Server Error, dapat %d", resp.StatusCode)
		}
	})
}

func TestAdminController_UploadImage(t *testing.T) {
	t.Run("Uji upload file multipart form-data dan verifikasi path URL /uploads/...", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupAdminControllerTest(mockRepo, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("image", "beethoven_concert.png")
		if err != nil {
			t.Fatalf("Gagal membuat multipart form file: %v", err)
		}
		fakeImageContent := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
		if _, err := part.Write(fakeImageContent); err != nil {
			t.Fatalf("Gagal menulis isi file gambar: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("Gagal menutup multipart writer: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/v1/admin/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}

		var apiResp models.APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			t.Fatalf("Gagal mendecode response: %v", err)
		}
		if !apiResp.Success {
			t.Fatalf("Ekspektasi success bernilai true")
		}

		dataMap, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Format payload data tidak valid: %v", apiResp.Data)
		}

		fileURL, ok := dataMap["url"].(string)
		if !ok || fileURL == "" {
			t.Fatalf("URL gambar tidak ditemukan pada response")
		}

		// Verifikasi path URL selalu diawali dengan skema HTTP dan berisi path /uploads/
		if !strings.Contains(fileURL, "/uploads/") {
			t.Fatalf("Ekspektasi URL mengandung '/uploads/', dapat: %s", fileURL)
		}
		if !strings.HasSuffix(fileURL, ".png") {
			t.Fatalf("Ekspektasi URL berakhiran '.png', dapat: %s", fileURL)
		}

		// Membersihkan file uji yang dibuat di folder uploads
		parts := strings.Split(fileURL, "/uploads/")
		if len(parts) == 2 {
			savedFileName := parts[1]
			savedFilePath := filepath.Clean(filepath.Join("uploads", savedFileName))
			_ = os.Remove(savedFilePath)
		}
	})

	t.Run("Gagal (400 Bad Request) saat file gambar tidak disertakan", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupAdminControllerTest(mockRepo, nil)

		req := httptest.NewRequest("POST", "/api/v1/admin/upload", bytes.NewBuffer([]byte{}))
		req.Header.Set("Content-Type", "multipart/form-data")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})
}
