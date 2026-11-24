package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

func TestLifecycleBroadcastE2E(t *testing.T) {
	var mu sync.Mutex
	eventState := models.EventItem{
		ID:       "evt-e2e-1",
		Title:    "Malam Simfoni Nusantara",
		Date:     time.Now().Add(3 * time.Hour).Format("2006-01-02"),
		Time:     time.Now().Add(3 * time.Hour).Format("15:04"),
		Status:   "SCHEDULED",
		IsClosed: false,
	}
	orderState := models.OrderRecord{
		ID:          "ord-e2e-1",
		OrderCode:   "SYM-E2E-100",
		UserID:      "usr-e2e-1",
		EventID:     eventState.ID,
		EventStatus: eventState.Status,
		EventTitle:  eventState.Title,
		Date:        eventState.Date,
		EventTime:   eventState.Time,
		UserName:    "Clara Wijaya",
		UserEmail:   "clara@example.com",
		Quantity:    1,
		TotalPrice:  750000,
		Status:      "ISSUED",
	}
	broadcastStore := make([]models.EmailBroadcastRecord, 0)

	eventRepo := &repositories.MockEventRepository{
		GetEventByIDFunc: func(ctx context.Context, id string) (*models.EventItem, error) {
			mu.Lock()
			defer mu.Unlock()
			copyEvt := eventState
			return &copyEvt, nil
		},
		CancelEventFunc: func(ctx context.Context, id string, reason string) (*models.EventItem, error) {
			mu.Lock()
			defer mu.Unlock()
			eventState.Status = "CANCELLED"
			eventState.IsClosed = true
			eventState.CancellationReason = reason
			orderState.EventStatus = "CANCELLED"
			copyEvt := eventState
			return &copyEvt, nil
		},
		ToggleEventCloseFunc: func(ctx context.Context, id string) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			if eventState.Status == "CANCELLED" {
				return false, repositories.ErrEventCancelled
			}
			eventState.IsClosed = !eventState.IsClosed
			return eventState.IsClosed, nil
		},
	}

	orderRepo := &repositories.MockOrderRepository{
		CreateOrderFunc: func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
			mu.Lock()
			defer mu.Unlock()
			if eventState.Status == "CANCELLED" {
				return nil, repositories.ErrEventCancelled
			}
			copyOrd := orderState
			return &copyOrd, nil
		},
		GetOrderByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
			mu.Lock()
			defer mu.Unlock()
			copyOrd := orderState
			return &copyOrd, nil
		},
		CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
			mu.Lock()
			defer mu.Unlock()
			if eventState.Status == "CANCELLED" {
				return nil, repositories.ErrEventCancelled
			}
			copyOrd := orderState
			copyOrd.Status = "CHECKED_IN"
			return &copyOrd, nil
		},
	}

	refundRepo := &repositories.MockRefundRepository{
		SubmitDirectUserRefundFunc: func(ctx context.Context, order *models.OrderRecord, input *models.SubmitUserRefundInput) (*models.RefundRequestRecord, error) {
			return &models.RefundRequestRecord{
				ID:           "ref-e2e-1",
				OrderCode:    order.OrderCode,
				UserEmail:    order.UserEmail,
				RefundAmount: order.TotalPrice,
				Status:       "PENDING",
			}, nil
		},
	}

	broadcastRepo := &repositories.MockBroadcastRepository{
		ResolveRecipientsFunc: func(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error) {
			return []models.BroadcastRecipient{
				{Name: orderState.UserName, Email: orderState.UserEmail, OrderCode: orderState.OrderCode},
			}, nil
		},
		CreateBroadcastFunc: func(ctx context.Context, record *models.EmailBroadcastRecord) error {
			mu.Lock()
			defer mu.Unlock()
			broadcastStore = append(broadcastStore, *record)
			return nil
		},
		GetAllBroadcastsFunc: func(ctx context.Context) ([]models.EmailBroadcastRecord, error) {
			mu.Lock()
			defer mu.Unlock()
			out := make([]models.EmailBroadcastRecord, len(broadcastStore))
			copy(out, broadcastStore)
			return out, nil
		},
	}

	app := fiber.New()
	eventCtrl := controllers.NewEventController(eventRepo)
	eventCtrl.SetBroadcastRepo(broadcastRepo)
	orderCtrl := controllers.NewOrderController(orderRepo)
	ticketCtrl := controllers.NewTicketController(orderRepo)
	refundCtrl := controllers.NewRefundControllerWithOrderRepo(refundRepo, orderRepo)
	broadcastCtrl := controllers.NewBroadcastController(broadcastRepo, eventRepo)

	api := app.Group("/api/v1")
	api.Post("/orders", orderCtrl.CreateOrder)
	api.Post("/user/orders/:orderCode/refund", middleware.RequireUserAuth, refundCtrl.SubmitUserRefund)
	api.Post("/admin/events/:id/cancel", eventCtrl.CancelEvent)
	api.Patch("/admin/events/:id/toggle-close", eventCtrl.ToggleEventClose)
	api.Post("/admin/tickets/checkin", ticketCtrl.CheckInTicket)
	api.Get("/admin/broadcasts", broadcastCtrl.GetAllBroadcasts)
	api.Post("/admin/broadcasts", broadcastCtrl.CreateBroadcast)

	// 1. Admin membatalkan konser + auto-notify
	cancelBody, _ := json.Marshal(models.CancelEventRequest{
		Reason:          "Pemeliharaan darurat gedung konser",
		NotifyAttendees: true,
	})
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/events/evt-e2e-1/cancel", bytes.NewBuffer(cancelBody))
	req1.Header.Set("Content-Type", "application/json")
	resp1, _ := app.Test(req1, -1)
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("Step 1 (CancelEvent) ekspektasi 200, dapat %d", resp1.StatusCode)
	}

	// 2. Admin mencoba membuka kembali konser yang sudah dibatalkan -> ditolak 400
	req2 := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/events/evt-e2e-1/toggle-close", nil)
	resp2, _ := app.Test(req2, -1)
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("Step 2 (ToggleEventClose on CANCELLED) ekspektasi 400, dapat %d", resp2.StatusCode)
	}

	// 3. Pembeli mencoba membeli tiket untuk konser yang dibatalkan -> ditolak 400
	orderBody, _ := json.Marshal(models.CreateOrderRequest{
		EventID:          "evt-e2e-1",
		TicketCategoryID: "cat-1",
		Quantity:         1,
		UserName:         "Budi",
		UserEmail:        "budi@example.com",
	})
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(orderBody))
	req3.Header.Set("Content-Type", "application/json")
	resp3, _ := app.Test(req3, -1)
	if resp3.StatusCode != http.StatusBadRequest {
		t.Fatalf("Step 3 (CreateOrder on CANCELLED) ekspektasi 400, dapat %d", resp3.StatusCode)
	}

	// 4. Petugas gate mencoba check-in tiket konser batal meski BypassOpenGate true -> ditolak 400
	checkInBody, _ := json.Marshal(models.CheckInRequest{Code: "SYM-E2E-100", BypassOpenGate: true})
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tickets/checkin", bytes.NewBuffer(checkInBody))
	req4.Header.Set("Content-Type", "application/json")
	resp4, _ := app.Test(req4, -1)
	if resp4.StatusCode != http.StatusBadRequest {
		t.Fatalf("Step 4 (CheckInTicket on CANCELLED) ekspektasi 400, dapat %d", resp4.StatusCode)
	}

	// 5. Pemilik tiket mengajukan refund (< 24 jam sebelum konser) -> bypass berhasil 201 Created
	userToken, _ := utils.GenerateUserToken("usr-e2e-1", "clara@example.com", "Clara Wijaya")
	refundBody, _ := json.Marshal(models.SubmitUserRefundInput{
		BankName:      "BCA",
		AccountNumber: "1234567890",
		AccountHolder: "Clara Wijaya",
		Reason:        "Konser dibatalkan penyelenggara",
	})
	req5 := httptest.NewRequest(http.MethodPost, "/api/v1/user/orders/SYM-E2E-100/refund", bytes.NewBuffer(refundBody))
	req5.Header.Set("Content-Type", "application/json")
	req5.Header.Set("Authorization", "Bearer "+userToken)
	resp5, _ := app.Test(req5, -1)
	if resp5.StatusCode != http.StatusCreated {
		t.Fatalf("Step 5 (SubmitUserRefund bypass 24h) ekspektasi 201, dapat %d", resp5.StatusCode)
	}

	// 6. Admin mengirim broadcast promo dan memeriksa riwayat
	bcastBody, _ := json.Marshal(models.CreateBroadcastRequest{
		TargetScope: "ALL_USERS",
		Category:    "PROMO",
		Subject:     "Kompensasi Promo Spesial",
		Headline:    "Diskon Konser Berikutnya",
		BodyText:    "Halo {{name}}, gunakan kode promo berikut.",
		PromoCode:   "SYMPHONIA10",
	})
	req6 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/broadcasts", bytes.NewBuffer(bcastBody))
	req6.Header.Set("Content-Type", "application/json")
	resp6, _ := app.Test(req6, -1)
	if resp6.StatusCode != http.StatusAccepted {
		t.Fatalf("Step 6 (CreateBroadcast) ekspektasi 202, dapat %d", resp6.StatusCode)
	}

	req7 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/broadcasts", nil)
	resp7, _ := app.Test(req7, -1)
	if resp7.StatusCode != http.StatusOK {
		t.Fatalf("Step 7 (GetAllBroadcasts) ekspektasi 200, dapat %d", resp7.StatusCode)
	}
	var listResp struct {
		Success bool                          `json:"success"`
		Data    []models.EmailBroadcastRecord `json:"data"`
	}
	_ = json.NewDecoder(resp7.Body).Decode(&listResp)
	if len(listResp.Data) != 2 {
		t.Fatalf("Ekspektasi 2 riwayat broadcast (auto-cancel + promo), dapat %d", len(listResp.Data))
	}
}
