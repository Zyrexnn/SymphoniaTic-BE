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

func setupBroadcastControllerTest(bRepo repositories.BroadcastRepository, eRepo repositories.EventRepository) *fiber.App {
	app := fiber.New()
	ctrl := controllers.NewBroadcastController(bRepo, eRepo)
	api := app.Group("/api/v1")
	api.Get("/admin/broadcasts", ctrl.GetAllBroadcasts)
	api.Post("/admin/broadcasts", ctrl.CreateBroadcast)
	api.Get("/admin/broadcasts/:id/recipients", ctrl.GetBroadcastRecipients)
	api.Post("/admin/broadcasts/:id/retry", ctrl.RetryFailedBroadcast)
	api.Get("/unsubscribe", ctrl.UnsubscribePromo)
	api.Post("/unsubscribe", ctrl.UnsubscribePromo)
	return app
}

func TestBroadcastController(t *testing.T) {
	t.Run("GET /admin/broadcasts mengembalikan 200 OK", func(t *testing.T) {
		mockRepo := &repositories.MockBroadcastRepository{
			GetAllBroadcastsFunc: func(ctx context.Context) ([]models.EmailBroadcastRecord, error) {
				return []models.EmailBroadcastRecord{
					{ID: "bcast-1", TargetScope: "ALL_USERS", Category: "ANNOUNCEMENT", Subject: "Info", Status: "COMPLETED"},
				}, nil
			},
		}
		app := setupBroadcastControllerTest(mockRepo, &repositories.MockEventRepository{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/broadcasts", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("POST /admin/broadcasts ALL_USERS mengembalikan 202 Accepted", func(t *testing.T) {
		var savedRecord *models.EmailBroadcastRecord
		mockRepo := &repositories.MockBroadcastRepository{
			ResolveRecipientsFunc: func(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error) {
				return []models.BroadcastRecipient{
					{Name: "Budi", Email: "budi@example.com"},
					{Name: "Siti", Email: "siti@example.com"},
				}, nil
			},
			CreateBroadcastFunc: func(ctx context.Context, record *models.EmailBroadcastRecord) error {
				savedRecord = record
				return nil
			},
		}
		app := setupBroadcastControllerTest(mockRepo, &repositories.MockEventRepository{})
		bodyBytes, _ := json.Marshal(models.CreateBroadcastRequest{
			TargetScope: "ALL_USERS",
			Category:    "PROMO",
			Subject:     "Promo Akhir Tahun",
			Headline:    "Potongan 10% Tiket Simfoni",
			BodyText:    "Halo {{name}}, gunakan kode promo di bawah ini.",
			PromoCode:   "SYMPHONIA10",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/broadcasts", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim request: %v", err)
		}
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("Ekspektasi 202 Accepted, dapat %d", resp.StatusCode)
		}
		if savedRecord == nil || savedRecord.TotalRecipients != 2 {
			t.Fatalf("Record broadcast tidak tersimpan dengan jumlah penerima 2: %+v", savedRecord)
		}
	})

	t.Run("POST /admin/broadcasts EVENT_ATTENDEES tanpa targetEventId mengembalikan 400 Bad Request", func(t *testing.T) {
		app := setupBroadcastControllerTest(&repositories.MockBroadcastRepository{}, &repositories.MockEventRepository{})
		bodyBytes, _ := json.Marshal(models.CreateBroadcastRequest{
			TargetScope: "EVENT_ATTENDEES",
			Category:    "ANNOUNCEMENT",
			Subject:     "Info Gate",
			Headline:    "Pembukaan Gate",
			BodyText:    "Gate dibuka pukul 18:00 WIB",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/broadcasts", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("POST /admin/broadcasts dengan penerima kosong mengembalikan 400 Bad Request", func(t *testing.T) {
		mockRepo := &repositories.MockBroadcastRepository{
			ResolveRecipientsFunc: func(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error) {
				return []models.BroadcastRecipient{}, nil
			},
		}
		app := setupBroadcastControllerTest(mockRepo, &repositories.MockEventRepository{})
		bodyBytes, _ := json.Marshal(models.CreateBroadcastRequest{
			TargetScope: "ALL_USERS",
			Category:    "ANNOUNCEMENT",
			Subject:     "Info",
			Headline:    "Judul",
			BodyText:    "Isi pesan",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/broadcasts", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("POST /admin/broadcasts CATEGORY_ATTENDEES tanpa targetCategoryId mengembalikan 400 Bad Request", func(t *testing.T) {
		app := setupBroadcastControllerTest(&repositories.MockBroadcastRepository{}, &repositories.MockEventRepository{})
		bodyBytes, _ := json.Marshal(models.CreateBroadcastRequest{
			TargetScope:   "CATEGORY_ATTENDEES",
			TargetEventID: "evt-1",
			Category:      "ANNOUNCEMENT",
			Subject:       "Info VVIP",
			Headline:      "Akses Lounge VVIP",
			BodyText:      "Silakan masuk melalui pintu timur",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/broadcasts", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Ekspektasi 400 Bad Request, dapat %d", resp.StatusCode)
		}
	})

	t.Run("POST /admin/broadcasts PROMO meneruskan excludePromoOptOut=true", func(t *testing.T) {
		capturedOptOut := false
		mockRepo := &repositories.MockBroadcastRepository{
			ResolveRecipientsFilteredFunc: func(ctx context.Context, scope, eventID, categoryID string, excludePromoOptOut bool) ([]models.BroadcastRecipient, error) {
				capturedOptOut = excludePromoOptOut
				return []models.BroadcastRecipient{{Name: "Budi", Email: "budi@example.com", UnsubscribeToken: "tok-1"}}, nil
			},
		}
		app := setupBroadcastControllerTest(mockRepo, &repositories.MockEventRepository{})
		bodyBytes, _ := json.Marshal(models.CreateBroadcastRequest{
			TargetScope: "ALL_USERS",
			Category:    "PROMO",
			Subject:     "Promo Khusus",
			Headline:    "Diskon Tiket",
			BodyText:    "Halo {{name}}",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/broadcasts", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("Ekspektasi 202 Accepted, dapat %d", resp.StatusCode)
		}
		if !capturedOptOut {
			t.Fatalf("Ekspektasi excludePromoOptOut=true untuk kategori PROMO")
		}
	})

	t.Run("GET /admin/broadcasts/:id/recipients mengembalikan daftar penerima", func(t *testing.T) {
		mockRepo := &repositories.MockBroadcastRepository{
			GetBroadcastRecipientsFunc: func(ctx context.Context, broadcastID string) ([]models.BroadcastRecipientRecord, error) {
				return []models.BroadcastRecipientRecord{
					{ID: "r-1", BroadcastID: broadcastID, RecipientEmail: "fail@example.com", Status: "FAILED", ErrorMessage: "timeout"},
				}, nil
			},
		}
		app := setupBroadcastControllerTest(mockRepo, &repositories.MockEventRepository{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/broadcasts/bcast-1/recipients", nil)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
	})

	t.Run("POST /admin/broadcasts/:id/retry mengirim ulang penerima FAILED", func(t *testing.T) {
		mockRepo := &repositories.MockBroadcastRepository{
			GetBroadcastByIDFunc: func(ctx context.Context, id string) (*models.EmailBroadcastRecord, error) {
				return &models.EmailBroadcastRecord{
					ID:          id,
					TargetScope: "ALL_USERS",
					Category:    "ANNOUNCEMENT",
					Subject:     "Info",
					Headline:    "Judul",
					BodyText:    "Isi",
					SentCount:   1,
				}, nil
			},
			GetFailedRecipientsFunc: func(ctx context.Context, broadcastID string) ([]models.BroadcastRecipient, error) {
				return []models.BroadcastRecipient{{Name: "Andi", Email: "andi@example.com"}}, nil
			},
		}
		app := setupBroadcastControllerTest(mockRepo, &repositories.MockEventRepository{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/broadcasts/bcast-1/retry", nil)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("Ekspektasi 202 Accepted, dapat %d", resp.StatusCode)
		}
	})

	t.Run("GET /unsubscribe memproses token valid dan menolak token tidak dikenal", func(t *testing.T) {
		mockRepo := &repositories.MockBroadcastRepository{
			UnsubscribeByTokenFunc: func(ctx context.Context, token string) error {
				if token == "valid-tok" {
					return nil
				}
				return repositories.ErrNotFound
			},
		}
		app := setupBroadcastControllerTest(mockRepo, &repositories.MockEventRepository{})

		respOK, _ := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/unsubscribe?token=valid-tok", nil), -1)
		if respOK.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK untuk token valid, dapat %d", respOK.StatusCode)
		}

		resp404, _ := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/unsubscribe?token=invalid-tok", nil), -1)
		if resp404.StatusCode != http.StatusNotFound {
			t.Fatalf("Ekspektasi 404 Not Found untuk token tidak valid, dapat %d", resp404.StatusCode)
		}
	})
}
