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

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
)

func TestTicketController_LookupEdgeCases(t *testing.T) {
	t.Run("Gagal (500 Internal Server Error) saat repository lookup gagal", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			LookupTicketByCodeFunc: func(ctx context.Context, code string) (*models.OrderRecord, error) {
				return nil, errors.New("db timeout")
			},
		}

		app := setupOrderControllerTest(mockRepo)
		req := httptest.NewRequest("GET", "/api/v1/tickets/lookup?code=SYM-ERR500", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("Ekspektasi 500 Internal Server Error, dapat %d", resp.StatusCode)
		}
	})
}

func TestTicketController_CheckInEdgeCases(t *testing.T) {
	t.Run("Sukses (200 OK) menerima kode dari field order_code dengan spasi", func(t *testing.T) {
		var receivedCode string
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				receivedCode = orderCode
				return &models.OrderRecord{
					ID:        "ord-ordercode",
					OrderCode: orderCode,
					Status:    "CHECKED_IN",
				}, nil
			},
		}

		app := setupOrderControllerTest(mockRepo)
		payload := models.CheckInRequest{OrderCode: "  SYM-ORDERCODE  "}
		bodyBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		if receivedCode != "SYM-ORDERCODE" {
			t.Fatalf("Ekspektasi kode ter-trim 'SYM-ORDERCODE', dapat %q", receivedCode)
		}
	})

	t.Run("Sukses (200 OK) fallback menerima kode dari query parameter", func(t *testing.T) {
		var receivedCode string
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				receivedCode = orderCode
				return &models.OrderRecord{
					ID:        "ord-query",
					OrderCode: orderCode,
					Status:    "CHECKED_IN",
				}, nil
			},
		}

		app := setupOrderControllerTest(mockRepo)
		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin?code=SYM-QUERYCODE", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		if receivedCode != "SYM-QUERYCODE" {
			t.Fatalf("Ekspektasi kode 'SYM-QUERYCODE', dapat %q", receivedCode)
		}
	})

	t.Run("Gagal (400 Bad Request) ErrTicketAlreadyUsed dengan fallback UserName", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					OrderCode: "SYM-USED-USER",
					UserName:  "Andi Wijaya",
				}, repositories.ErrTicketAlreadyUsed
			},
		}

		app := setupOrderControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.CheckInRequest{Code: "SYM-USED-USER"})
		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
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
		if !strings.Contains(apiResp.Message, "Andi Wijaya") {
			t.Fatalf("Ekspektasi pesan fallback memuat UserName, dapat %q", apiResp.Message)
		}
	})

	t.Run("Gagal (400 Bad Request) ketika status tiket belum ISSUED (ErrTicketNotIssued)", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return nil, repositories.ErrTicketNotIssued
			},
		}

		app := setupOrderControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.CheckInRequest{Code: "SYM-PENDING"})
		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
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
		if apiResp.Message != "Status tiket tidak valid untuk check-in" {
			t.Fatalf("Pesan respons tidak sesuai: %q", apiResp.Message)
		}
	})

	t.Run("Gagal (500 Internal Server Error) saat error database tak terduga", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return nil, errors.New("unexpected db error")
			},
		}

		app := setupOrderControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.CheckInRequest{Code: "SYM-ERR"})
		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("Ekspektasi 500 Internal Server Error, dapat %d", resp.StatusCode)
		}
	})

	t.Run("Menyesuaikan totalCount saat Quantity lebih kecil dari CheckedInSeats", func(t *testing.T) {
		mockRepo := &repositories.MockOrderRepository{
			CheckInTicketFunc: func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
				return &models.OrderRecord{
					OrderCode:      "SYM-QTY-FIX",
					Quantity:       0,
					CheckedInSeats: "A-1",
					Status:         "ISSUED",
				}, nil
			},
		}

		app := setupOrderControllerTest(mockRepo)
		bodyBytes, _ := json.Marshal(models.CheckInRequest{Code: "SYM-QTY-FIX"})
		req := httptest.NewRequest("POST", "/api/v1/admin/tickets/checkin", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Gagal mengirim HTTP request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Ekspektasi 200 OK, dapat %d", resp.StatusCode)
		}
		var apiResp models.APIResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiResp)
		if apiResp.Message != "Check-in kursi berhasil (1/1 tiket telah masuk)" {
			t.Fatalf("Pesan respons tidak sesuai: %q", apiResp.Message)
		}
	})
}
