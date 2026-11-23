package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

func setupOrderControllerTest(mockRepo *repositories.MockOrderRepository) *fiber.App {
	app := fiber.New()
	orderCtrl := controllers.NewOrderController(mockRepo)
	ticketCtrl := controllers.NewTicketController(mockRepo)

	api := app.Group("/api/v1")
	api.Post("/orders", orderCtrl.CreateOrder)
	api.Get("/admin/orders", orderCtrl.GetAllOrders)
	api.Patch("/admin/orders/:id/status", orderCtrl.UpdateOrderStatus)
	api.Get("/tickets/lookup", ticketCtrl.LookupTicketByCode)
	api.Post("/admin/tickets/checkin", ticketCtrl.CheckInTicket)
	api.Get("/admin/tickets/attendance-summary", ticketCtrl.GetAttendanceSummary)

	return app
}

func TestOrderController_CreateOrder(t *testing.T) {
	t.Run("Sukses (201 Created) saat kuota mencukupi", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:         "ord-success-01",
					OrderCode:  "SYM-100001",
					UserEmail:  req.UserEmail,
					UserName:   req.UserName,
					Quantity:   req.Quantity,
					TotalPrice: 500000,
					Status:     "ISSUED",
					CreatedAt:  time.Now(),
				}, nil
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CreateOrderRequest{
			UserName:         "Budi Santoso",
			UserEmail:        "budi@example.com",
			TicketCategoryID: "cat-vip-01",
			Quantity:         2,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Status tidak sesuai: ekspektasi 201 Created, dapat %d", resp.StatusCode)
		}

		var apiResp models.APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			t.Fatalf("Gagal mendecode response body: %v", err)
		}
		if !apiResp.Success {
			t.Fatalf("Ekspektasi success bernilai true")
		}
	})

	t.Run("Gagal (400 Bad Request) saat kuota habis", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				return nil, repositories.ErrQuotaExceeded
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CreateOrderRequest{
			UserName:         "Budi Santoso",
			UserEmail:        "budi@example.com",
			TicketCategoryID: "cat-vip-01",
			Quantity:         2,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat event ditutup", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				return nil, repositories.ErrEventClosed
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CreateOrderRequest{
			UserName:         "Budi Santoso",
			UserEmail:        "budi@example.com",
			TicketCategoryID: "cat-vip-01",
			Quantity:         2,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat quantity kurang dari 1", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CreateOrderRequest{
			UserName:         "Budi Santoso",
			UserEmail:        "budi@example.com",
			TicketCategoryID: "cat-vip-01",
			Quantity:         0,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat quantity lebih dari 4", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CreateOrderRequest{
			UserName:         "Budi Santoso",
			UserEmail:        "budi@example.com",
			TicketCategoryID: "cat-vip-01",
			Quantity:         5,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat field wajib kosong", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CreateOrderRequest{
			UserName: "",
			Quantity: 2,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(bodyBytes))
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

func TestOrderController_GetAllOrders(t *testing.T) {
	t.Run("Sukses (200 OK) mengembalikan pagination dan daftar order", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			GetAllOrdersFunc: func(ctx context.Context, searchQuery, statusFilter string) ([]models.OrderRecord, error) {
				return []models.OrderRecord{
					{
						ID:         "ord-01",
						OrderCode:  "SYM-100001",
						UserEmail:  "user1@example.com",
						UserName:   "User Satu",
						Status:     "ISSUED",
						TotalPrice: 350000,
					},
					{
						ID:         "ord-02",
						OrderCode:  "SYM-100002",
						UserEmail:  "user2@example.com",
						UserName:   "User Dua",
						Status:     "CHECKED_IN",
						TotalPrice: 500000,
					},
				}, nil
			},
		}

		app := setupOrderControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/admin/orders?search=User&status=ISSUED", nil)
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
}

func TestOrderController_UpdateOrderStatus(t *testing.T) {
	t.Run("Sukses (200 OK) saat status diubah ke CHECKED_IN", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			UpdateOrderStatusFunc: func(ctx context.Context, id, status string) error {
				if id == "ord-01" && status == "CHECKED_IN" {
					return nil
				}
				return repositories.ErrNotFound
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.UpdateOrderStatusRequest{
			Status: "CHECKED_IN",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/api/v1/admin/orders/ord-01/status", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Sukses (200 OK) saat status diubah ke REMINDED dan memicu notifikasi", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			UpdateOrderStatusFunc: func(ctx context.Context, id, status string) error {
				return nil
			},
			GetOrderByIDFunc: func(ctx context.Context, id string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:           "ord-01",
					OrderCode:    "SYM-100001",
					UserEmail:    "user1@example.com",
					UserName:     "User Satu",
					EventTitle:   "Simfoni No. 5",
					Venue:        "Aula Simfonia Jakarta",
					VenueAddress: "Jl. Industri Blok B14 No.1, Kemayoran",
					Date:         "2026-12-15 @ 19:30 WIB",
					EventTime:    "19:30 WIB",
					OpenGate:     "18:00 WIB",
				}, nil
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.UpdateOrderStatusRequest{
			Status: "REMINDED",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/api/v1/admin/orders/ord-01/status", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (404 Not Found) saat pesanan tidak ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			UpdateOrderStatusFunc: func(ctx context.Context, id, status string) error {
				return repositories.ErrNotFound
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.UpdateOrderStatusRequest{
			Status: "CHECKED_IN",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/api/v1/admin/orders/ord-404/status", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat status kosong", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupOrderControllerTest(mockRepo)

		payload := models.UpdateOrderStatusRequest{
			Status: "",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PATCH", "/api/v1/admin/orders/ord-01/status", bytes.NewBuffer(bodyBytes))
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

func TestOrderController_LookupTicketByCode(t *testing.T) {
	t.Run("Sukses (200 OK) saat kode ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			LookupTicketByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				if code == "SYM-VALID-CODE" {
					return &models.OrderRecord{
						ID:         "ord-01",
						OrderCode:  "SYM-VALID-CODE",
						UserEmail:  "valid@example.com",
						UserName:   "Valid User",
						EventTitle: "Simfoni No. 9 Beethoven",
						Status:     "ISSUED",
					}, nil
				}
				return nil, repositories.ErrNotFound
			},
		}

		app := setupOrderControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/tickets/lookup?code=SYM-VALID-CODE", nil)
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

	t.Run("Gagal (404 Not Found) saat kode tidak ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			LookupTicketByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return nil, repositories.ErrNotFound
			},
		}

		app := setupOrderControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/tickets/lookup?code=SYM-NOTFOUND", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat kode kosong", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupOrderControllerTest(mockRepo)

		req := httptest.NewRequest("GET", "/api/v1/tickets/lookup", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})
}

func TestTicketController_CheckInTicket(t *testing.T) {
	t.Run("Sukses (200 OK) saat tiket valid dan status ISSUED", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:        "ord-01",
					OrderCode: "SYM-987654",
					Status:    "CHECKED_IN",
				}, nil
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{
			Code: "SYM-987654",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
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

	t.Run("Gagal (400 Bad Request) saat tiket sudah pernah digunakan", func(t *testing.T) {
		prevScanAt := time.Date(2026, 11, 22, 18, 30, 0, 0, time.UTC)
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:                "ord-used",
					OrderCode:         "SYM-ALREADY-USED",
					UserName:          "Budi",
					CheckedInSeat:     "A-1",
					CheckedInAttendee: "Siti",
					CheckedInAt:       &prevScanAt,
				}, repositories.ErrTicketAlreadyUsed
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{
			Code: "SYM-ALREADY-USED-A-1",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
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
		if !strings.Contains(apiResp.Message, "Siti") || !strings.Contains(apiResp.Message, "A-1") {
			t.Fatalf("Pesan respons seharusnya memuat nama pemilik kursi dan nomor kursi, dapat: %s", apiResp.Message)
		}
		dataMap, ok := apiResp.Data.(map[string]interface{})
		if !ok || dataMap["checkedInAttendee"] != "Siti" || dataMap["checkedInAt"] == nil {
			t.Fatalf("Payload data pada ErrTicketAlreadyUsed seharusnya menyertakan checkedInAttendee & checkedInAt, dapat: %+v", apiResp.Data)
		}
	})

	t.Run("Gagal (400 Bad Request) saat tiket dibatalkan atau direfund", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return nil, repositories.ErrTicketInvalid
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{
			Code: "SYM-REFUNDED",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
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
		if apiResp.Message != "Tiket tidak valid karena telah dibatalkan atau di-refund" {
			t.Fatalf("Pesan respons tidak sesuai: dapat %s", apiResp.Message)
		}
	})

	t.Run("Gagal (404 Not Found) saat tiket tidak ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return nil, repositories.ErrNotFound
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{
			Code: "SYM-NOTFOUND",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat kode tiket kosong", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CheckInRequest{
			Code: "",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) saat konser ditutup oleh panitia", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return nil, repositories.ErrEventClosed
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{
			Code: "SYM-CLOSED",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
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
		if apiResp.Message != "Konser telah ditutup oleh panitia dan tiket tidak dapat digunakan untuk check-in" {
			t.Fatalf("Pesan respons tidak sesuai: dapat %s", apiResp.Message)
		}
	})

	t.Run("Gagal (400 Bad Request) saat gerbang masuk konser belum dibuka", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return nil, repositories.ErrEventNotStarted
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{
			Code: "SYM-NOT-STARTED",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
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
		if apiResp.Message != "Gerbang masuk (Open Gate) belum dibuka untuk konser ini" {
			t.Fatalf("Pesan respons tidak sesuai: dapat %s", apiResp.Message)
		}
	})

	t.Run("Gagal (400 Bad Request) saat acara konser telah selesai (expired)", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return nil, repositories.ErrEventExpired
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{
			Code: "SYM-EXPIRED",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
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
		if apiResp.Message != "Tiket tidak dapat digunakan karena acara konser telah selesai" {
			t.Fatalf("Pesan respons tidak sesuai: dapat %s", apiResp.Message)
		}
	})
}

func TestTicketController_GetAttendanceSummary(t *testing.T) {
	t.Run("Sukses (200 OK) mengambil ringkasan kehadiran gate", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			GetAttendanceSummaryFunc: func(ctx context.Context, eventID string) (*models.GateAttendanceSummary, error) {
				if eventID != "evt-1" {
					t.Errorf("Ekspektasi eventID evt-1, dapat %s", eventID)
				}
				return &models.GateAttendanceSummary{
					TotalTicketsIssued: 100,
					TotalCheckedIn:     45,
					RemainingCheckIn:   55,
					CheckInPercentage:  45.0,
				}, nil
			},
		}

		app := setupOrderControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/admin/tickets/attendance-summary?event_id=evt-1", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}

		var apiResp struct {
			Success bool                         `json:"success"`
			Message string                       `json:"message"`
			Data    models.GateAttendanceSummary `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			t.Fatalf("Gagal decode JSON response: %v", err)
		}
		if !apiResp.Success {
			t.Fatalf("Ekspektasi success true, dapat false")
		}
		if apiResp.Data.TotalTicketsIssued != 100 || apiResp.Data.TotalCheckedIn != 45 || apiResp.Data.RemainingCheckIn != 55 {
			t.Fatalf("Data ringkasan kehadiran tidak sesuai: %+v", apiResp.Data)
		}
	})

	t.Run("Gagal (500 Internal Server Error) jika database error", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			GetAttendanceSummaryFunc: func(ctx context.Context, eventID string) (*models.GateAttendanceSummary, error) {
				return nil, errors.New("koneksi database terputus")
			},
		}

		app := setupOrderControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/admin/tickets/attendance-summary", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("Status tidak sesuai: ekspektasi 500 Internal Server Error, dapat %d", resp.StatusCode)
		}
	})
}

func TestOrderController_CreateOrderSeatErrors(t *testing.T) {
	t.Run("Gagal (409 Conflict) ketika kursi sudah dipesan orang lain", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				return nil, repositories.ErrSeatUnavailable
			},
		}
		app := setupOrderControllerTest(mockRepo)
		payload := models.CreateOrderRequest{
			EventID:          "evt-1",
			TicketCategoryID: "cat-vip",
			Quantity:         1,
			UserName:         "Budi",
			UserEmail:        "budi@example.com",
			SeatNumbers:      []string{"A-1"},
		}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("Ekspektasi 409 Conflict, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) ketika pilihan nomor kursi tidak valid", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				return nil, repositories.ErrInvalidSeatSelection
			},
		}
		app := setupOrderControllerTest(mockRepo)
		payload := models.CreateOrderRequest{
			EventID:          "evt-1",
			TicketCategoryID: "cat-vip",
			Quantity:         2,
			UserName:         "Budi",
			UserEmail:        "budi@example.com",
			SeatNumbers:      []string{"A-1"},
		}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Sukses (200 OK) partial check-in per kursi menampilkan progres jumlah masuk", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:             "ord-01",
					OrderCode:      "SYM-987654",
					Quantity:       2,
					SeatNumbers:    "A-1, A-2",
					CheckedInSeats: "A-1",
					Status:         "ISSUED",
				}, nil
			},
		}
		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{Code: "QR-SYM-987654-A-1"}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		var apiResp models.APIResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiResp)
		if !apiResp.Success || apiResp.Message != "Check-in kursi berhasil (1/2 tiket telah masuk)" {
			t.Fatalf("Pesan partial check-in tidak sesuai: %+v", apiResp)
		}
	})

	t.Run("Meneruskan flag bypassOpenGate ke repository saat check-in", func(t *testing.T) {
		var receivedBypass bool
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				receivedBypass = bypassOpenGate
				return &models.OrderRecord{
					ID:        "ord-bypass",
					OrderCode: "SYM-FUTURE",
					Status:    "CHECKED_IN",
				}, nil
			},
		}
		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{Code: "SYM-FUTURE", BypassOpenGate: true}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		if !receivedBypass {
			t.Fatalf("Flag bypassOpenGate tidak diteruskan ke repository")
		}
	})
}

func TestOrderController_GuestTicketLimitsAndPromo(t *testing.T) {
	t.Setenv("JWT_SECRET", "symphoniatic-test-jwt-secret-key-min-32")

	userToken, err := utils.GenerateUserToken("usr-member-1", "member@example.com", "USER")
	if err != nil {
		t.Fatalf("Gagal membuat token member: %v", err)
	}

	t.Run("Gagal (400) ketika guest memesan lebih dari 2 tiket", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CreateOrderRequest{
			UserName:         "Tamu Konser",
			UserEmail:        "tamu@example.com",
			TicketCategoryID: "cat-1",
			Quantity:         3,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request untuk guest > 2 tiket, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400) ketika guest mencoba menggunakan kode promo khusus member", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				if authenticatedUserID == "" && req.PromoCode == "SYMPHONIA10" {
					return nil, repositories.ErrPromoMemberOnly
				}
				return &models.OrderRecord{ID: "ord-unexpected", Status: "ISSUED"}, nil
			},
		}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CreateOrderRequest{
			UserName:         "Tamu Promo",
			UserEmail:        "tamu@example.com",
			TicketCategoryID: "cat-1",
			Quantity:         1,
			PromoCode:        "SYMPHONIA10",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request untuk guest dengan promo khusus member, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Sukses (201) ketika guest memesan maksimal 2 tiket dengan promo publik", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					ID:             "ord-guest-pub",
					OrderCode:      "SYM-PUB01",
					UserName:       req.UserName,
					UserEmail:      req.UserEmail,
					Quantity:       req.Quantity,
					TotalPrice:     200000,
					PromoCode:      req.PromoCode,
					DiscountAmount: 25000,
					Status:         "ISSUED",
					CreatedAt:      time.Now(),
				}, nil
			},
		}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CreateOrderRequest{
			UserName:         "Tamu Publik",
			UserEmail:        "tamu@example.com",
			TicketCategoryID: "cat-1",
			Quantity:         2,
			PromoCode:        "PUBLIK25K",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Ekspektasi 201 Created untuk guest dengan promo publik, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Sukses (201) ketika member memesan 4 tiket dengan kode promo", func(t *testing.T) {
		var capturedPromo string
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				capturedPromo = req.PromoCode
				return &models.OrderRecord{
					ID:             "ord-member-1",
					OrderCode:      "SYM-MEM001",
					UserID:         authenticatedUserID,
					UserName:       req.UserName,
					UserEmail:      req.UserEmail,
					Quantity:       req.Quantity,
					TotalPrice:     350000,
					PromoCode:      req.PromoCode,
					DiscountAmount: 50000,
					Status:         "ISSUED",
					CreatedAt:      time.Now(),
				}, nil
			},
		}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CreateOrderRequest{
			UserName:         "Anggota Setia",
			UserEmail:        "member@example.com",
			TicketCategoryID: "cat-1",
			Quantity:         4,
			PromoCode:        "harmony50k",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Ekspektasi 201 Created untuk member 4 tiket + promo, dapat %d", resp.StatusCode)
		}
		if capturedPromo != "HARMONY50K" {
			t.Fatalf("Ekspektasi PromoCode ternormalisasi HARMONY50K, dapat %q", capturedPromo)
		}
	})

	t.Run("Gagal (400) ketika repository menolak promo yang sudah dipakai atau tidak valid", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
				return nil, repositories.ErrPromoAlreadyUsed
			},
		}
		app := setupOrderControllerTest(mockRepo)

		payload := models.CreateOrderRequest{
			UserName:         "Anggota Setia",
			UserEmail:        "member@example.com",
			TicketCategoryID: "cat-1",
			Quantity:         2,
			PromoCode:        "SYMPHONIA10",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})
}
