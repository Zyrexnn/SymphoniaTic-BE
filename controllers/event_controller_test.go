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

func setupEventControllerTest(mockRepo *repositories.MockEventRepository) *fiber.App {
	app := fiber.New()
	eventCtrl := controllers.NewEventController(mockRepo)
	categoryCtrl := controllers.NewCategoryController(mockRepo)

	api := app.Group("/api/v1")
	api.Get("/events", eventCtrl.GetEvents)
	api.Get("/events/:id", eventCtrl.GetEventByID)
	api.Get("/events/:id/seats", eventCtrl.GetEventSeats)
	api.Post("/admin/events", eventCtrl.CreateEvent)
	api.Put("/admin/events/:id", eventCtrl.UpdateEvent)
	api.Delete("/admin/events/:id", eventCtrl.DeleteEvent)
	api.Patch("/admin/events/:id/toggle-close", eventCtrl.ToggleEventClose)
	api.Post("/admin/events/:id/cancel", eventCtrl.CancelEvent)
	api.Post("/admin/events/:id/reschedule", eventCtrl.RescheduleEvent)

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
			OpenGate:    "17:30",
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

	t.Run("Gagal (400 Bad Request) saat OpenGate kosong", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{}
		app := setupEventControllerTest(mockRepo)

		payload := models.CreateEventRequest{
			Title:  "Konser Sore",
			Artist: "Jakarta Ensemble",
			Venue:  "Aula Simfonia",
			Date:   "2026-12-30",
			Time:   "14:21 WIB",
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

	t.Run("Gagal (400 Bad Request) saat OpenGate lebih lambat atau sama dengan Time", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{}
		app := setupEventControllerTest(mockRepo)

		cases := []struct {
			openGate string
			time     string
		}{
			{"18:00 WIB", "14:21 WIB"},
			{"14:21 WIB", "14:21 WIB"},
			{"15.00 WIB", "02:21 PM"},
		}

		for _, tc := range cases {
			payload := models.CreateEventRequest{
				Title:    "Konser Siang",
				Artist:   "Jakarta Ensemble",
				Venue:    "Aula Simfonia",
				Date:     "2026-12-30",
				Time:     tc.time,
				OpenGate: tc.openGate,
			}
			bodyBytes, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/v1/admin/events", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Gagal mengirim HTTP request: %v", err)
			}

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("Status tidak sesuai untuk openGate=%s time=%s: ekspektasi 400, dapat %d", tc.openGate, tc.time, resp.StatusCode)
			}

			var apiResp models.APIResponse
			_ = json.NewDecoder(resp.Body).Decode(&apiResp)
			if apiResp.Message != "Waktu Open Gate harus lebih awal dari waktu mulai pertunjukan konser" {
				t.Fatalf("Pesan error tidak sesuai: dapat %q", apiResp.Message)
			}
		}
	})

	t.Run("Gagal (400 Bad Request) saat item rundown tidak lengkap", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{}
		app := setupEventControllerTest(mockRepo)

		payload := models.CreateEventRequest{
			Title:    "Konser Siang",
			Artist:   "Jakarta Ensemble",
			Venue:    "Aula Simfonia",
			Date:     "2026-12-30",
			Time:     "14:21 WIB",
			OpenGate: "13:21 WIB",
			Rundown: []models.RundownItem{
				{Time: "13:21 WIB", Activity: ""},
			},
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

	t.Run("Sukses (201 Created) menerima salesEndAt sebelum konser dimulai", func(t *testing.T) {
		var capturedSalesEndAt *time.Time
		mockRepo := &repositories.MockEventRepository{
			CreateEventFunc: func(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error) {
				capturedSalesEndAt = req.SalesEndAt
				return &models.EventItem{ID: "evt-cutoff", Title: req.Title, SalesEndAt: req.SalesEndAt}, nil
			},
		}
		app := setupEventControllerTest(mockRepo)
		wib := time.FixedZone("WIB", 7*3600)
		cutoff := time.Date(2026, 12, 30, 17, 0, 0, 0, wib)

		payload := models.CreateEventRequest{
			Title:      "Simfoni Natal",
			Artist:     "Jakarta Ensemble",
			Venue:      "Aula Simfonia",
			Date:       "2026-12-30",
			Time:       "19:30 WIB",
			OpenGate:   "18:00 WIB",
			SalesEndAt: &cutoff,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/events", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Ekspektasi 201 Created, dapat %d", resp.StatusCode)
		}
		if capturedSalesEndAt == nil || !capturedSalesEndAt.Equal(cutoff) {
			t.Fatalf("SalesEndAt tidak diteruskan ke repository: dapat %v", capturedSalesEndAt)
		}
	})

	t.Run("Gagal (400 Bad Request) saat salesEndAt melebihi waktu mulai konser", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{}
		app := setupEventControllerTest(mockRepo)
		wib := time.FixedZone("WIB", 7*3600)
		lateCutoff := time.Date(2026, 12, 30, 21, 0, 0, 0, wib)

		payload := models.CreateEventRequest{
			Title:      "Simfoni Natal",
			Artist:     "Jakarta Ensemble",
			Venue:      "Aula Simfonia",
			Date:       "2026-12-30",
			Time:       "19:30 WIB",
			OpenGate:   "18:00 WIB",
			SalesEndAt: &lateCutoff,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/events", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
		var apiResp models.APIResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiResp)
		if apiResp.Message != "Batas waktu penjualan tiket tidak boleh melebihi waktu mulai konser" {
			t.Fatalf("Pesan error tidak sesuai: dapat %q", apiResp.Message)
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
			Title:    "Judul Konser Baru Diperbarui",
			Venue:    "Aula Simfonia Lantai 2",
			Date:     "Minggu, 18 April 2027",
			Time:     "19:30 WIB",
			OpenGate: "18:00 WIB",
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

	t.Run("Gagal (400 Bad Request) saat OpenGate kosong atau >= Time", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{}
		app := setupEventControllerTest(mockRepo)

		for _, payload := range []models.UpdateEventRequest{
			{Title: "Tanpa Open Gate", Time: "14:21 WIB", OpenGate: ""},
			{Title: "Open Gate Terlambat", Time: "14:21 WIB", OpenGate: "18:00 WIB"},
		} {
			bodyBytes, _ := json.Marshal(payload)
			req := httptest.NewRequest("PUT", "/api/v1/admin/events/evt-01", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Gagal mengirim HTTP request: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("Status tidak sesuai untuk %q: ekspektasi 400, dapat %d", payload.Title, resp.StatusCode)
			}
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
			Title:    "Judul Baru",
			Date:     "Minggu, 18 April 2027",
			Time:     "19:30 WIB",
			OpenGate: "18:00 WIB",
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

	t.Run("Gagal (400 Bad Request) saat UpdateEvent menetapkan salesEndAt melewati konser dimulai", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{}
		app := setupEventControllerTest(mockRepo)
		wib := time.FixedZone("WIB", 7*3600)
		lateCutoff := time.Date(2026, 12, 30, 20, 0, 0, 0, wib)

		payload := models.UpdateEventRequest{
			Title:      "Judul Baru",
			Date:       "2026-12-30",
			Time:       "19:30 WIB",
			OpenGate:   "18:00 WIB",
			SalesEndAt: &lateCutoff,
		}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("PUT", "/api/v1/admin/events/evt-01", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("ToggleEventClose Gagal (400 Bad Request) jika konser sudah dibatalkan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			ToggleEventCloseFunc: func(ctx context.Context, id string) (bool, error) {
				return false, repositories.ErrEventCancelled
			},
		}
		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("PATCH", "/api/v1/admin/events/evt-1/toggle-close", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("CancelEvent dengan NotifyAttendees true mencatat dan mengirim broadcast otomatis", func(t *testing.T) {
		mockEventRepo := &repositories.MockEventRepository{
			CancelEventFunc: func(ctx context.Context, id string, reason string) (*models.EventItem, error) {
				return &models.EventItem{
					ID:                 id,
					Title:              "Simfoni Malam",
					Date:               "Sabtu, 18 April 2027",
					Status:             "CANCELLED",
					IsClosed:           true,
					CancellationReason: reason,
				}, nil
			},
		}
		var createdBroadcast *models.EmailBroadcastRecord
		mockBroadcastRepo := &repositories.MockBroadcastRepository{
			ResolveRecipientsFunc: func(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error) {
				return []models.BroadcastRecipient{
					{Name: "Clara", Email: "clara@example.com", OrderCode: "SYM-101"},
				}, nil
			},
			CreateBroadcastFunc: func(ctx context.Context, record *models.EmailBroadcastRecord) error {
				createdBroadcast = record
				return nil
			},
		}

		app := fiber.New()
		eventCtrl := controllers.NewEventController(mockEventRepo)
		eventCtrl.SetBroadcastRepo(mockBroadcastRepo)
		app.Post("/api/v1/admin/events/:id/cancel", eventCtrl.CancelEvent)

		bodyBytes, _ := json.Marshal(models.CancelEventRequest{
			Reason:          "Force majeure venue",
			NotifyAttendees: true,
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		if createdBroadcast == nil || createdBroadcast.Category != "EVENT_CANCELLED" || createdBroadcast.TotalRecipients != 1 {
			t.Fatalf("Broadcast otomatis pembatalan tidak tercatat dengan benar: %+v", createdBroadcast)
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

	t.Run("Gagal (400 Bad Request) saat konser memiliki transaksi tiket aktif", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			DeleteEventFunc: func(ctx context.Context, id string) error {
				return repositories.ErrEventHasOrders
			},
		}

		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("DELETE", "/api/v1/admin/events/evt-with-orders", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Status tidak sesuai: ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}

		var apiResp models.APIResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiResp)
		if apiResp.Message != "Konser tidak dapat dihapus karena sudah memiliki transaksi tiket aktif. Gunakan fitur Tutup Order." {
			t.Fatalf("Pesan respons tidak sesuai: dapat %q", apiResp.Message)
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

	t.Run("Gagal (400 Bad Request) ketika mengurangi kuota kategori di bawah jumlah kursi terjual", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			UpdateCategoryFunc: func(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error {
				return repositories.ErrQuotaBelowBooked
			},
		}
		app := setupEventControllerTest(mockRepo)
		payload := models.UpdateTicketCategoryRequest{Name: "VIP", Price: 750000, Quota: 1}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest("PUT", "/api/v1/admin/categories/cat-01", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400 Bad Request) ketika menghapus kategori yang memiliki kursi terjual", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			DeleteCategoryFunc: func(ctx context.Context, catID string) error {
				return repositories.ErrCategoryHasBookedSeats
			},
		}
		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("DELETE", "/api/v1/admin/categories/cat-01", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("ToggleEventClose Gagal (400 Bad Request) jika konser sudah dibatalkan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			ToggleEventCloseFunc: func(ctx context.Context, id string) (bool, error) {
				return false, repositories.ErrEventCancelled
			},
		}
		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("PATCH", "/api/v1/admin/events/evt-1/toggle-close", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("CancelEvent dengan NotifyAttendees true mencatat dan mengirim broadcast otomatis", func(t *testing.T) {
		mockEventRepo := &repositories.MockEventRepository{
			CancelEventFunc: func(ctx context.Context, id string, reason string) (*models.EventItem, error) {
				return &models.EventItem{
					ID:                 id,
					Title:              "Simfoni Malam",
					Date:               "Sabtu, 18 April 2027",
					Status:             "CANCELLED",
					IsClosed:           true,
					CancellationReason: reason,
				}, nil
			},
		}
		var createdBroadcast *models.EmailBroadcastRecord
		mockBroadcastRepo := &repositories.MockBroadcastRepository{
			ResolveRecipientsFunc: func(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error) {
				return []models.BroadcastRecipient{
					{Name: "Clara", Email: "clara@example.com", OrderCode: "SYM-101"},
				}, nil
			},
			CreateBroadcastFunc: func(ctx context.Context, record *models.EmailBroadcastRecord) error {
				createdBroadcast = record
				return nil
			},
		}

		app := fiber.New()
		eventCtrl := controllers.NewEventController(mockEventRepo)
		eventCtrl.SetBroadcastRepo(mockBroadcastRepo)
		app.Post("/api/v1/admin/events/:id/cancel", eventCtrl.CancelEvent)

		bodyBytes, _ := json.Marshal(models.CancelEventRequest{
			Reason:          "Force majeure venue",
			NotifyAttendees: true,
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		if createdBroadcast == nil || createdBroadcast.Category != "EVENT_CANCELLED" || createdBroadcast.TotalRecipients != 1 {
			t.Fatalf("Broadcast otomatis pembatalan tidak tercatat dengan benar: %+v", createdBroadcast)
		}
	})
}

func TestEventController_GetEventSeats(t *testing.T) {
	t.Run("Sukses (200 OK) mengembalikan daftar kursi konser", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			GetSeatsByEventIDFunc: func(ctx context.Context, eventID string) ([]models.SeatItem, error) {
				return []models.SeatItem{
					{ID: "s1", EventID: eventID, CategoryID: "cat-1", SeatCode: "A-1", RowLabel: "A", SeatNumber: 1, Status: "AVAILABLE"},
					{ID: "s2", EventID: eventID, CategoryID: "cat-1", SeatCode: "A-2", RowLabel: "A", SeatNumber: 2, Status: "BOOKED"},
				}, nil
			},
		}
		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/events/evt-1/seats", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (404 Not Found) jika konser tidak ditemukan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			GetSeatsByEventIDFunc: func(ctx context.Context, eventID string) ([]models.SeatItem, error) {
				return nil, repositories.ErrNotFound
			},
		}
		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/events/evt-404/seats", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Ekspektasi 404 Not Found, dapat %d", resp.StatusCode)
		}
	})
}

func TestEventController_LifecycleActions(t *testing.T) {
	t.Run("CancelEvent Sukses (200 OK)", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			CancelEventFunc: func(ctx context.Context, id string, reason string) (*models.EventItem, error) {
				return &models.EventItem{
					ID:                 id,
					Status:             "CANCELLED",
					IsClosed:           true,
					CancellationReason: reason,
				}, nil
			},
		}
		app := setupEventControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.CancelEventRequest{Reason: "Kondisi force majeure di gedung pertunjukan"})
		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("CancelEvent Gagal (400 Bad Request) jika alasan kosong", func(t *testing.T) {
		app := setupEventControllerTest(&repositories.MockEventRepository{})
		bodyBytes, _ := json.Marshal(models.CancelEventRequest{Reason: "   "})
		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("RescheduleEvent Sukses (200 OK)", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			RescheduleEventFunc: func(ctx context.Context, id string, req *models.RescheduleEventRequest) (*models.EventItem, error) {
				return &models.EventItem{
					ID:               id,
					Status:           "RESCHEDULED",
					Date:             req.NewDate,
					Time:             req.NewTime,
					OriginalDate:     "Sabtu, 18 April 2027",
					OriginalTime:     "19:30 WIB",
					RescheduleReason: req.Reason,
				}, nil
			},
		}
		app := setupEventControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.RescheduleEventRequest{
			NewDate: "Sabtu, 25 April 2027",
			NewTime: "20:00 WIB",
			Reason:  "Penyesuaian jadwal konduktor tamu",
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/reschedule", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("RescheduleEvent Gagal (400 Bad Request) jika konser sudah dibatalkan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			RescheduleEventFunc: func(ctx context.Context, id string, req *models.RescheduleEventRequest) (*models.EventItem, error) {
				return nil, repositories.ErrEventCancelled
			},
		}
		app := setupEventControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.RescheduleEventRequest{
			NewDate: "Sabtu, 25 April 2027",
			NewTime: "20:00 WIB",
			Reason:  "Penyesuaian jadwal",
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/reschedule", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("RescheduleEvent Gagal (400 Bad Request) jika tanggal/waktu/openGate tidak valid", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{}
		app := setupEventControllerTest(mockRepo)

		invalidPayloads := []models.RescheduleEventRequest{
			{NewDate: "Tanggal Rahasia", NewTime: "20:00 WIB", Reason: "Test"},
			{NewDate: "2027-04-25", NewTime: "99:99", Reason: "Test"},
			{NewDate: "2027-04-25", NewTime: "20:00 WIB", NewOpenGate: "21:00 WIB", Reason: "Test"},
		}
		for _, p := range invalidPayloads {
			bodyBytes, _ := json.Marshal(p)
			req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/reschedule", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Gagal mengirim request: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("Ekspektasi 400 untuk payload %+v, dapat %d", p, resp.StatusCode)
			}
		}
	})

	t.Run("RescheduleEvent menormalkan tanggal dan jam sebelum dikirim ke repository", func(t *testing.T) {
		var captured *models.RescheduleEventRequest
		mockRepo := &repositories.MockEventRepository{
			RescheduleEventFunc: func(ctx context.Context, id string, req *models.RescheduleEventRequest) (*models.EventItem, error) {
				captured = req
				return &models.EventItem{
					ID:     id,
					Status: "RESCHEDULED",
					Date:   req.NewDate,
					Time:   req.NewTime,
				}, nil
			},
		}
		app := setupEventControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.RescheduleEventRequest{
			NewDate: "2027-04-25",
			NewTime: "8.00 PM",
			Reason:  "Penyesuaian jadwal",
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/reschedule", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		if captured == nil || captured.NewDate != "Minggu, 25 April 2027" || captured.NewTime != "20:00 WIB" {
			t.Fatalf("Hasil normalisasi tidak sesuai: %+v", captured)
		}
	})

	t.Run("CreateEvent menormalkan tanggal ISO ke format kanonik Indonesia", func(t *testing.T) {
		var captured *models.CreateEventRequest
		mockRepo := &repositories.MockEventRepository{
			CreateEventFunc: func(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error) {
				captured = req
				return &models.EventItem{
					ID:       "evt-norm",
					Title:    req.Title,
					Date:     req.Date,
					Time:     req.Time,
					OpenGate: req.OpenGate,
				}, nil
			},
		}
		app := setupEventControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.CreateEventRequest{
			Title:    "Konser Akhir Tahun",
			Artist:   "Jakarta Philharmonic",
			Venue:    "Aula Simfonia",
			Date:     "2026-12-30",
			Time:     "19:30 WIB",
			OpenGate: "18:00 WIB",
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/events", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Ekspektasi 201 Created, dapat %d", resp.StatusCode)
		}
		if captured == nil || captured.Date != "Rabu, 30 Desember 2026" {
			t.Fatalf("Tanggal tidak dinormalisasi kanonik: %+v", captured)
		}
	})

	t.Run("ToggleEventClose Gagal (400 Bad Request) jika konser sudah dibatalkan", func(t *testing.T) {
		mockRepo := &repositories.MockEventRepository{
			ToggleEventCloseFunc: func(ctx context.Context, id string) (bool, error) {
				return false, repositories.ErrEventCancelled
			},
		}
		app := setupEventControllerTest(mockRepo)
		req := httptest.NewRequest("PATCH", "/api/v1/admin/events/evt-1/toggle-close", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("CancelEvent dengan NotifyAttendees true mencatat dan mengirim broadcast otomatis", func(t *testing.T) {
		mockEventRepo := &repositories.MockEventRepository{
			CancelEventFunc: func(ctx context.Context, id string, reason string) (*models.EventItem, error) {
				return &models.EventItem{
					ID:                 id,
					Title:              "Simfoni Malam",
					Date:               "Sabtu, 18 April 2027",
					Status:             "CANCELLED",
					IsClosed:           true,
					CancellationReason: reason,
				}, nil
			},
		}
		var createdBroadcast *models.EmailBroadcastRecord
		mockBroadcastRepo := &repositories.MockBroadcastRepository{
			ResolveRecipientsFunc: func(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error) {
				return []models.BroadcastRecipient{
					{Name: "Clara", Email: "clara@example.com", OrderCode: "SYM-101"},
				}, nil
			},
			CreateBroadcastFunc: func(ctx context.Context, record *models.EmailBroadcastRecord) error {
				createdBroadcast = record
				return nil
			},
		}

		app := fiber.New()
		eventCtrl := controllers.NewEventController(mockEventRepo)
		eventCtrl.SetBroadcastRepo(mockBroadcastRepo)
		app.Post("/api/v1/admin/events/:id/cancel", eventCtrl.CancelEvent)

		bodyBytes, _ := json.Marshal(models.CancelEventRequest{
			Reason:          "Force majeure venue",
			NotifyAttendees: true,
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/events/evt-1/cancel", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		if createdBroadcast == nil || createdBroadcast.Category != "EVENT_CANCELLED" || createdBroadcast.TotalRecipients != 1 {
			t.Fatalf("Broadcast otomatis pembatalan tidak tercatat dengan benar: %+v", createdBroadcast)
		}
	})
}
