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
					ID:         "ord-01",
					OrderCode:  "SYM-100001",
					UserEmail:  "user1@example.com",
					UserName:   "User Satu",
					EventTitle: "Simfoni No. 5",
					Venue:      "Aula Simfonia Jakarta",
					Date:       "2026-12-15",
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
			CheckInTicketFunc: func(ctx context.Context, orderCode string) (*models.OrderRecord, error) {
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
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string) (*models.OrderRecord, error) {
				return nil, repositories.ErrTicketAlreadyUsed
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{
			Code: "SYM-ALREADY-USED",
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
		if apiResp.Message != "Tiket ini sudah pernah digunakan untuk check-in sebelumnya" {
			t.Fatalf("Pesan respons tidak sesuai: dapat %s", apiResp.Message)
		}
	})

	t.Run("Gagal (400 Bad Request) saat tiket dibatalkan atau direfund", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string) (*models.OrderRecord, error) {
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
			CheckInTicketFunc: func(ctx context.Context, orderCode string) (*models.OrderRecord, error) {
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
}
