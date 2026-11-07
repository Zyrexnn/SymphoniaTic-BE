package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

func setupEventControllerTest(mockRepo *repositories.MockEventRepository) *fiber.App {
	app := fiber.New()
	eventCtrl := controllers.NewEventController(mockRepo)
	categoryCtrl := controllers.NewCategoryController(mockRepo)

	api := app.Group("/api/v1")
	api.Get("/events", eventCtrl.GetEvents)
	api.Get("/events/:id", eventCtrl.GetEventByID)
	api.Post("/admin/events", eventCtrl.CreateEvent)
	api.Put("/admin/events/:id", eventCtrl.UpdateEvent)
	api.Delete("/admin/events/:id", eventCtrl.DeleteEvent)
	api.Patch("/admin/events/:id/toggle-close", eventCtrl.ToggleEventClose)

	api.Post("/admin/events/:id/categories", categoryCtrl.CreateTicketCategory)
	api.Put("/admin/categories/:id", categoryCtrl.UpdateTicketCategory)
	api.Delete("/admin/categories/:id", categoryCtrl.DeleteTicketCategory)

	return app
}

func TestEventController_GetEvents(t *testing.T) {
	t.Run("Sukses (200 OK) mengembalikan daftar event", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			GetAllEventsFunc: func(ctx context.Context) ([]models.EventItem, error) {
				return []models.EventItem{
					{
						ID:          "evt-01",
						Title:       "Malam Mahakarya Beethoven",
						Artist:      "Jakarta Philharmonic",
						Venue:       "Aula Simfonia Jakarta",
						Date:        "2026-12-20",
						Time:        "19:30",
						Category:    "SIMFONI UTAMA",
						Description: "Pertunjukan simfoni klasik legendaris.",
					},
					{
						ID:          "evt-02",
						Title:       "Konser Chopin Nocturne",
						Artist:      "Nusantara Chamber Ensemble",
						Venue:       "Usmar Ismail Hall",
						Date:        "2026-12-25",
						Time:        "20:00",
						Category:    "RESITAL SOLO",
						Description: "Malam resital piano klasik romantis.",
					},
				}, nil
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/events", nil)
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

func TestEventController_GetEventByID(t *testing.T) {
	t.Run("Sukses (200 OK) saat ID ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			GetEventByIDFunc: func(ctx context.Context, id string) (*models.EventItem, error) {
				if id == "evt-01" {
					return &models.EventItem{
						ID:     "evt-01",
						Title:  "Malam Mahakarya Beethoven",
						Artist: "Jakarta Philharmonic",
						Venue:  "Aula Simfonia Jakarta",
						Date:   "2026-12-20",
						Time:   "19:30",
					}, nil
				}
				return nil, repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/events/evt-01", nil)
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

	t.Run("Gagal (404 Not Found) saat ID tidak ada", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			GetEventByIDFunc: func(ctx context.Context, id string) (*models.EventItem, error) {
				return nil, repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/events/evt-non-existent", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})
}

func TestEventController_CreateEvent(t *testing.T) {
	t.Run("Sukses (201 Created) validasi payload dan response", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			CreateEventFunc: func(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error) {
				return &models.EventItem{
					ID:          "evt-new-01",
					Title:       req.Title,
					Artist:      req.Artist,
					Venue:       req.Venue,
					Date:        req.Date,
					Time:        req.Time,
					Category:    req.Category,
					Description: req.Description,
				}, nil
			},
		}

		app := setupEventControllerTest(mockRepo)
		payload := models.CreateEventRequest{
			Title:       "Konser Musim Dingin Vivaldi",
			Artist:      "Solo String Quartet",
			Venue:       "Gedung Kesenian Jakarta",
			Date:        "2026-12-30",
			Time:        "19:00",
			Category:    "SIMFONI UTAMA",
			Description: "Interpretasi Four Seasons karya Antonio Vivaldi.",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/events", bytes.NewBuffer(bodyBytes))
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

	t.Run("Gagal (400 Bad Request) saat field wajib kosong", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{}
		app := setupEventControllerTest(mockRepo)

		payload := models.CreateEventRequest{
			Title: "",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/events", bytes.NewBuffer(bodyBytes))
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

func TestEventController_UpdateEvent(t *testing.T) {
	t.Run("Sukses (200 OK) validasi payload dan response", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			UpdateEventFunc: func(ctx context.Context, id string, req *models.UpdateEventRequest) error {
				if id == "evt-01" {
					return nil
				}
				return repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		payload := models.UpdateEventRequest{
			Title: "Judul Konser Baru Diperbarui",
			Venue: "Aula Simfonia Lantai 2",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PUT", "/api/v1/admin/events/evt-01", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (404 Not Found) saat event tidak ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			UpdateEventFunc: func(ctx context.Context, id string, req *models.UpdateEventRequest) error {
				return repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		payload := models.UpdateEventRequest{
			Title: "Judul Baru",
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PUT", "/api/v1/admin/events/evt-404", bytes.NewBuffer(bodyBytes))
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

func TestEventController_ToggleEventClose(t *testing.T) {
	t.Run("Sukses (200 OK) mengubah status penutupan konser", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			ToggleEventCloseFunc: func(ctx context.Context, id string) (bool, error) {
				if id == "evt-01" {
					return true, nil
				}
				return false, repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("PATCH", "/api/v1/admin/events/evt-01/toggle-close", nil)
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

	t.Run("Gagal (404 Not Found) saat event tidak ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			ToggleEventCloseFunc: func(ctx context.Context, id string) (bool, error) {
				return false, repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("PATCH", "/api/v1/admin/events/evt-404/toggle-close", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})
}

func TestEventController_DeleteEvent(t *testing.T) {
	t.Run("Sukses (200 OK) menghapus konser", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			DeleteEventFunc: func(ctx context.Context, id string) error {
				if id == "evt-01" {
					return nil
				}
				return repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("DELETE", "/api/v1/admin/events/evt-01", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (404 Not Found) saat event tidak ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			DeleteEventFunc: func(ctx context.Context, id string) error {
				return repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("DELETE", "/api/v1/admin/events/evt-404", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Status tidak sesuai: ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})
}

func TestCategoryController_Management(t *testing.T) {
	t.Run("Sukses (201 Created) membuat kategori tiket baru", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			AddCategoryFunc: func(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error) {
				return &models.TicketCategory{
					ID:             "cat-new-01",
					EventID:        eventID,
					Name:           req.Name,
					Price:          req.Price,
					Quota:          req.Quota,
					RemainingQuota: req.Quota,
				}, nil
			},
		}

		app := setupEventControllerTest(mockRepo)
		payload := models.CreateCategoryInput{
			Name:  "VVIP Diamond",
			Price: 1500000,
			Quota: 50,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-01/categories", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Status tidak sesuai: ekspektasi 201 Created, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Sukses (200 OK) memperbarui kategori tiket", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			UpdateCategoryFunc: func(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error {
				if catID == "cat-01" {
					return nil
				}
				return repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		payload := models.UpdateTicketCategoryRequest{
			Name:  "VVIP Diamond Updated",
			Price: 1750000,
			Quota: 60,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PUT", "/api/v1/admin/categories/cat-01", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Sukses (200 OK) menghapus kategori tiket", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			DeleteCategoryFunc: func(ctx context.Context, catID string) error {
				if catID == "cat-01" {
					return nil
				}
				return repositories.ErrNotFound
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("DELETE", "/api/v1/admin/categories/cat-01", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status tidak sesuai: ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})
}
