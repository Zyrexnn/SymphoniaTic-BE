package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

func TestEventController(t *testing.T) {
	mockRepo := &repositories.MockEventRepository{}
	ctrl := controllers.NewEventController(mockRepo)

	app := fiber.New()
	app.Get("/events", ctrl.GetEvents)
	app.Get("/events/:id", ctrl.GetEventByID)
	app.Post("/admin/events", ctrl.CreateEvent)
	app.Put("/admin/events/:id", ctrl.UpdateEvent)
	app.Delete("/admin/events/:id", ctrl.DeleteEvent)
	app.Patch("/admin/events/:id/toggle-close", ctrl.ToggleEventClose)

	mockRepo.GetAllEventsFunc = func(ctx context.Context) ([]models.EventItem, error) {
		return []models.EventItem{
			{ID: "evt-1", Title: "Simfoni Klasik"},
		}, nil
	}

	req := httptest.NewRequest("GET", "/events", nil)
	resp, err := app.Test(req, -1)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GetEvents gagal: status %d", resp.StatusCode)
	}

	mockRepo.GetEventByIDFunc = func(ctx context.Context, id string) (*models.EventItem, error) {
		if id == "evt-1" {
			return &models.EventItem{ID: "evt-1", Title: "Simfoni Klasik"}, nil
		}
		return nil, repositories.ErrNotFound
	}

	req = httptest.NewRequest("GET", "/events/evt-1", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GetEventByID gagal: status %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/events/evt-404", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GetEventByID 404 gagal: status %d", resp.StatusCode)
	}

	mockRepo.CreateEventFunc = func(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error) {
		return &models.EventItem{
			ID:    "evt-new",
			Title: req.Title,
		}, nil
	}

	body, _ := json.Marshal(models.CreateEventRequest{
		Title:  "Konser Baru",
		Artist: "Orkestra Jakarta",
		Venue:  "Aula Simfonia",
		Date:   "2026-12-01",
		Time:   "19:30",
	})
	req = httptest.NewRequest("POST", "/admin/events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("CreateEvent gagal: status %d", resp.StatusCode)
	}

	mockRepo.UpdateEventFunc = func(ctx context.Context, id string, req *models.UpdateEventRequest) error {
		return nil
	}
	updateBody, _ := json.Marshal(models.UpdateEventRequest{Title: "Konser Update"})
	req = httptest.NewRequest("PUT", "/admin/events/evt-1", bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("UpdateEvent gagal: status %d", resp.StatusCode)
	}

	mockRepo.DeleteEventFunc = func(ctx context.Context, id string) error {
		return nil
	}
	req = httptest.NewRequest("DELETE", "/admin/events/evt-1", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DeleteEvent gagal: status %d", resp.StatusCode)
	}

	mockRepo.ToggleEventCloseFunc = func(ctx context.Context, id string) (bool, error) {
		return true, nil
	}
	req = httptest.NewRequest("PATCH", "/admin/events/evt-1/toggle-close", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ToggleEventClose gagal: status %d", resp.StatusCode)
	}
}

func TestCategoryController(t *testing.T) {
	mockRepo := &repositories.MockEventRepository{}
	ctrl := controllers.NewCategoryController(mockRepo)

	app := fiber.New()
	app.Post("/admin/events/:id/categories", ctrl.CreateTicketCategory)
	app.Put("/admin/categories/:id", ctrl.UpdateTicketCategory)
	app.Delete("/admin/categories/:id", ctrl.DeleteTicketCategory)

	mockRepo.AddCategoryFunc = func(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error) {
		return &models.TicketCategory{
			ID:    "cat-1",
			Name:  req.Name,
			Price: req.Price,
			Quota: req.Quota,
		}, nil
	}

	body, _ := json.Marshal(models.CreateCategoryInput{
		Name:  "VIP",
		Price: 500000,
		Quota: 100,
	})
	req := httptest.NewRequest("POST", "/admin/events/evt-1/categories", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("CreateTicketCategory gagal: status %d", resp.StatusCode)
	}

	mockRepo.UpdateCategoryFunc = func(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error {
		return nil
	}
	updateBody, _ := json.Marshal(models.UpdateTicketCategoryRequest{
		Name:  "VIP Updated",
		Price: 600000,
		Quota: 120,
	})
	req = httptest.NewRequest("PUT", "/admin/categories/cat-1", bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("UpdateTicketCategory gagal: status %d", resp.StatusCode)
	}

	mockRepo.DeleteCategoryFunc = func(ctx context.Context, catID string) error {
		return nil
	}
	req = httptest.NewRequest("DELETE", "/admin/categories/cat-1", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DeleteTicketCategory gagal: status %d", resp.StatusCode)
	}
}

func TestTicketController(t *testing.T) {
	mockRepo := &repositories.MockOrderRepository{}
	ctrl := controllers.NewTicketController(mockRepo)

	app := fiber.New()
	app.Get("/tickets/lookup", ctrl.LookupTicketByCode)

	mockRepo.LookupTicketByCodeFunc = func(ctx context.Context, code string) (*models.OrderRecord, error) {
		if code == "SYM-123456" {
			return &models.OrderRecord{OrderCode: "SYM-123456", EventTitle: "Simfoni No. 9"}, nil
		}
		return nil, repositories.ErrNotFound
	}

	req := httptest.NewRequest("GET", "/tickets/lookup?code=SYM-123456", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("LookupTicketByCode berhasil gagal: status %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/tickets/lookup?code=SYM-NOTFOUND", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("LookupTicketByCode tidak ditemukan gagal: status %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/tickets/lookup", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("LookupTicketByCode parameter kosong gagal: status %d", resp.StatusCode)
	}
}

func TestAdminController(t *testing.T) {
	mockRepo := &repositories.MockOrderRepository{}
	authCfg := &config.AuthConfig{
		AdminUsername:   "admin",
		AdminPassword:   "secret123",
		PetugasUsername: "petugas",
		PetugasPassword: "gate123",
	}
	ctrl := controllers.NewAdminController(mockRepo, authCfg)

	app := fiber.New()
	app.Post("/admin/login", ctrl.AdminLogin)
	app.Get("/admin/dashboard", ctrl.GetAdminDashboardMetrics)

	loginBody, _ := json.Marshal(models.AdminLoginRequest{
		Username: "admin",
		Password: "secret123",
	})
	req := httptest.NewRequest("POST", "/admin/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("AdminLogin gagal: status %d", resp.StatusCode)
	}

	invalidLogin, _ := json.Marshal(models.AdminLoginRequest{
		Username: "admin",
		Password: "wrongpassword",
	})
	req = httptest.NewRequest("POST", "/admin/login", bytes.NewBuffer(invalidLogin))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("AdminLogin invalid gagal: status %d", resp.StatusCode)
	}

	mockRepo.GetAdminDashboardMetricsFunc = func(ctx context.Context) (*models.AdminDashboardMetrics, error) {
		return &models.AdminDashboardMetrics{
			TotalRevenue: 10000000,
			TicketsSold:  50,
		}, nil
	}

	req = httptest.NewRequest("GET", "/admin/dashboard", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GetAdminDashboardMetrics gagal: status %d", resp.StatusCode)
	}
}

func TestRefundController(t *testing.T) {
	mockRefundRepo := &repositories.MockRefundRepository{}
	mockOrderRepo := &repositories.MockOrderRepository{}
	ctrl := controllers.NewRefundControllerWithOrderRepo(mockRefundRepo, mockOrderRepo)

	app := fiber.New()
	app.Post("/refunds/request-otp", ctrl.RequestRefundOTP)
	app.Post("/refunds/submit", ctrl.SubmitRefund)
	app.Post("/refunds/status", ctrl.GetRefundStatus)
	app.Get("/admin/refunds", ctrl.AdminGetAllRefunds)
	app.Patch("/admin/refunds/:id/status", ctrl.AdminUpdateRefundStatus)

	mockOrderRepo.GetOrderByCodeFunc = func(ctx context.Context, code string) (*models.OrderRecord, error) {
		if code == "SYM-123456" {
			return &models.OrderRecord{
				ID:         "ord-1",
				OrderCode:  "SYM-123456",
				UserEmail:  "user@example.com",
				Status:     "ISSUED",
				TotalPrice: 350000,
			}, nil
		}
		return nil, repositories.ErrNotFound
	}

	mockRefundRepo.RequestRefundOTPFunc = func(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error {
		return nil
	}

	otpReqBody, _ := json.Marshal(models.RequestRefundOTPInput{
		OrderCode: "SYM-123456",
		UserEmail: "user@example.com",
	})
	req := httptest.NewRequest("POST", "/refunds/request-otp", bytes.NewBuffer(otpReqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("RequestRefundOTP gagal: status %d", resp.StatusCode)
	}

	mockRefundRepo.SubmitRefundFunc = func(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error) {
		return &models.RefundRequestRecord{
			ID:           "rf-1",
			OrderCode:    input.OrderCode,
			RefundAmount: 350000,
			Status:       "PENDING",
		}, nil
	}

	submitBody, _ := json.Marshal(models.SubmitRefundInput{
		OrderCode:     "SYM-123456",
		UserEmail:     "user@example.com",
		OTPCode:       "123456",
		BankName:      "BCA",
		AccountNumber: "1234567890",
		AccountHolder: "Test User",
		Reason:        "Ada halangan acara",
	})
	req = httptest.NewRequest("POST", "/refunds/submit", bytes.NewBuffer(submitBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SubmitRefund gagal: status %d", resp.StatusCode)
	}

	mockRefundRepo.GetRefundStatusFunc = func(ctx context.Context, orderCode, userEmail string) (*models.RefundRequestRecord, string, error) {
		return &models.RefundRequestRecord{
			OrderCode: orderCode,
			Status:    "PENDING",
		}, "REFUND_REQUESTED", nil
	}

	statusReqBody, _ := json.Marshal(models.CheckRefundStatusRequest{
		OrderCode: "SYM-123456",
		UserEmail: "user@example.com",
	})
	req = httptest.NewRequest("POST", "/refunds/status", bytes.NewBuffer(statusReqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GetRefundStatus gagal: status %d", resp.StatusCode)
	}

	mockRefundRepo.GetAllRefundsFunc = func(ctx context.Context) ([]models.RefundRequestRecord, error) {
		return []models.RefundRequestRecord{
			{ID: "rf-1", OrderCode: "SYM-123456", Status: "PENDING"},
		}, nil
	}
	req = httptest.NewRequest("GET", "/admin/refunds", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("AdminGetAllRefunds gagal: status %d", resp.StatusCode)
	}

	mockRefundRepo.UpdateRefundStatusFunc = func(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error) {
		return &models.RefundRequestRecord{
			ID:           refundID,
			OrderCode:    "SYM-123456",
			UserEmail:    "user@example.com",
			RefundAmount: 350000,
			Status:       newStatus,
			AdminNote:    adminNote,
		}, nil
	}

	updateRefundBody, _ := json.Marshal(models.UpdateRefundStatusRequest{
		Status:    "APPROVED",
		AdminNote: "Disetujui",
	})
	req = httptest.NewRequest("PATCH", "/admin/refunds/rf-1/status", bytes.NewBuffer(updateRefundBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("AdminUpdateRefundStatus gagal: status %d", resp.StatusCode)
	}
}
