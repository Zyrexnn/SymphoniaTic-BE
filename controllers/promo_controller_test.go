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
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

func setupPromoControllerTest(mockPromoRepo *repositories.MockPromoRepository) *fiber.App {
	app := fiber.New()
	promoCtrl := controllers.NewPromoController(mockPromoRepo)
	api := app.Group("/api/v1")
	api.Get("/promos/active", promoCtrl.GetActivePromos)
	api.Post("/promos/validate", promoCtrl.ValidatePromo)

	admin := api.Group("/admin/promos")
	admin.Get("", promoCtrl.AdminGetAllPromos)
	admin.Post("", promoCtrl.AdminCreatePromo)
	admin.Put("/:id", promoCtrl.AdminUpdatePromo)
	admin.Patch("/:id/toggle-active", promoCtrl.AdminTogglePromoActive)
	admin.Delete("/:id", promoCtrl.AdminDeletePromo)
	return app
}

func TestPromoController_ValidatePromo(t *testing.T) {
	t.Setenv("JWT_SECRET", "symphoniatic-test-jwt-secret-key-min-32")

	validPromo := &models.Promotion{
		ID:             "promo-1",
		Code:           "SYMPHONIA10",
		Description:    "Diskon Anggota 10%",
		DiscountType:   "PERCENTAGE",
		DiscountVal:    10,
		MinOrderAmount: 100000,
		MaxDiscount:    50000,
		IsMemberOnly:   true,
		IsActive:       true,
		Quota:          100,
		RemainingQuota: 50,
		ValidFrom:      time.Now().Add(-1 * time.Hour),
		ValidUntil:     time.Now().Add(24 * time.Hour),
	}

	userToken, err := utils.GenerateUserToken("user-1", "member@example.com", "USER")
	if err != nil {
		t.Fatalf("Gagal membuat token user: %v", err)
	}

	t.Run("Gagal (400) jika promo sedang dinonaktifkan admin", func(t *testing.T) {
		inactive := *validPromo
		inactive.IsActive = false
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			GetPromoByCodeFunc: func(ctx context.Context, code string) (*models.Promotion, error) {
				return &inactive, nil
			},
		})
		body, _ := json.Marshal(models.ValidatePromoRequest{PromoCode: "SYMPHONIA10", Subtotal: 300000})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promos/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400) jika guest mencoba memakai promo khusus member", func(t *testing.T) {
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			GetPromoByCodeFunc: func(ctx context.Context, code string) (*models.Promotion, error) {
				return validPromo, nil
			},
		})
		body, _ := json.Marshal(models.ValidatePromoRequest{PromoCode: "SYMPHONIA10", Subtotal: 300000})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promos/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400) jika kode promo kedaluwarsa", func(t *testing.T) {
		expired := *validPromo
		expired.ValidUntil = time.Now().Add(-1 * time.Minute)
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			GetPromoByCodeFunc: func(ctx context.Context, code string) (*models.Promotion, error) {
				return &expired, nil
			},
		})
		body, _ := json.Marshal(models.ValidatePromoRequest{PromoCode: "SYMPHONIA10", Subtotal: 300000})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promos/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400) jika subtotal belum memenuhi minimal transaksi", func(t *testing.T) {
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			GetPromoByCodeFunc: func(ctx context.Context, code string) (*models.Promotion, error) {
				return validPromo, nil
			},
		})
		body, _ := json.Marshal(models.ValidatePromoRequest{PromoCode: "SYMPHONIA10", Subtotal: 50000})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promos/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Gagal (400) jika anggota sudah pernah memakai kode promo ini", func(t *testing.T) {
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			GetPromoByCodeFunc: func(ctx context.Context, code string) (*models.Promotion, error) {
				return validPromo, nil
			},
			HasUserUsedPromoFunc: func(ctx context.Context, promotionID, userID string) (bool, error) {
				return true, nil
			},
		})
		body, _ := json.Marshal(models.ValidatePromoRequest{PromoCode: "SYMPHONIA10", Subtotal: 300000})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promos/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Sukses (200) memvalidasi promo dan menghitung potongan untuk anggota", func(t *testing.T) {
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			GetPromoByCodeFunc: func(ctx context.Context, code string) (*models.Promotion, error) {
				return validPromo, nil
			},
			HasUserUsedPromoFunc: func(ctx context.Context, promotionID, userID string) (bool, error) {
				return false, nil
			},
		})
		body, _ := json.Marshal(models.ValidatePromoRequest{PromoCode: "symphonia10", Subtotal: 300000})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promos/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var apiResp struct {
			Success bool                         `json:"success"`
			Data    models.ValidatePromoResponse `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if !apiResp.Success || !apiResp.Data.Valid || apiResp.Data.DiscountAmount != 30000 || apiResp.Data.FinalTotal != 270000 {
			t.Fatalf("unexpected validation payload: %+v", apiResp)
		}
	})
}

func TestPromoController_AdminCRUDAndActiveFeed(t *testing.T) {
	now := time.Now()
	samplePayload := models.CreatePromoRequest{
		Code:           "beethoven25",
		Description:    "Promo Khusus Simfoni",
		DiscountType:   "percentage",
		DiscountVal:    25,
		MinOrderAmount: 150000,
		MaxDiscount:    100000,
		IsMemberOnly:   true,
		IsActive:       true,
		Quota:          200,
		ValidFrom:      now,
		ValidUntil:     now.Add(7 * 24 * time.Hour),
	}

	t.Run("Sukses (200) mengambil promo aktif publik", func(t *testing.T) {
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			GetActivePromosFunc: func(ctx context.Context) ([]models.Promotion, error) {
				return []models.Promotion{{ID: "p-1", Code: "BEETHOVEN25", IsActive: true}}, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/promos/active", nil)
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Sukses (201) admin membuat promo kustom dengan normalisasi kode huruf besar", func(t *testing.T) {
		var savedCode, savedType string
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			CreatePromoFunc: func(ctx context.Context, req *models.CreatePromoRequest) (*models.Promotion, error) {
				savedCode = req.Code
				savedType = req.DiscountType
				return &models.Promotion{
					ID:           "promo-custom-1",
					Code:         req.Code,
					DiscountType: req.DiscountType,
					DiscountVal:  req.DiscountVal,
					IsActive:     req.IsActive,
				}, nil
			},
		})
		body, _ := json.Marshal(samplePayload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/promos", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req, -1)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		if savedCode != "BEETHOVEN25" || savedType != "PERCENTAGE" {
			t.Fatalf("expected normalized BEETHOVEN25 / PERCENTAGE, got %q / %q", savedCode, savedType)
		}
	})

	t.Run("Gagal (400) jika persentase melebihi 100 atau kode duplikat", func(t *testing.T) {
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			CreatePromoFunc: func(ctx context.Context, req *models.CreatePromoRequest) (*models.Promotion, error) {
				return nil, repositories.ErrPromoDuplicateCode
			},
		})
		invalidPct := samplePayload
		invalidPct.DiscountVal = 150
		body1, _ := json.Marshal(invalidPct)
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/promos", bytes.NewBuffer(body1))
		req1.Header.Set("Content-Type", "application/json")
		if resp1, _ := app.Test(req1, -1); resp1.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for >100%%, got %d", resp1.StatusCode)
		}

		body2, _ := json.Marshal(samplePayload)
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/promos", bytes.NewBuffer(body2))
		req2.Header.Set("Content-Type", "application/json")
		if resp2, _ := app.Test(req2, -1); resp2.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for duplicate code, got %d", resp2.StatusCode)
		}
	})

	t.Run("Sukses (200) admin update, toggle active, dan hapus promo", func(t *testing.T) {
		app := setupPromoControllerTest(&repositories.MockPromoRepository{
			GetAllPromosFunc: func(ctx context.Context) ([]models.Promotion, error) {
				return []models.Promotion{{ID: "p-1", Code: "BEETHOVEN25"}}, nil
			},
			UpdatePromoFunc: func(ctx context.Context, id string, req *models.UpdatePromoRequest) (*models.Promotion, error) {
				return &models.Promotion{ID: id, Code: req.Code}, nil
			},
			TogglePromoActiveFunc: func(ctx context.Context, id string) (bool, error) {
				return false, nil
			},
			DeletePromoFunc: func(ctx context.Context, id string) error {
				return nil
			},
		})

		if resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/admin/promos", nil), -1); resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on GET all, got %d", resp.StatusCode)
		}

		body, _ := json.Marshal(samplePayload)
		putReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/promos/p-1", bytes.NewBuffer(body))
		putReq.Header.Set("Content-Type", "application/json")
		if resp, _ := app.Test(putReq, -1); resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on PUT, got %d", resp.StatusCode)
		}

		if resp, _ := app.Test(httptest.NewRequest(http.MethodPatch, "/api/v1/admin/promos/p-1/toggle-active", nil), -1); resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on PATCH toggle, got %d", resp.StatusCode)
		}

		if resp, _ := app.Test(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/promos/p-1", nil), -1); resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on DELETE, got %d", resp.StatusCode)
		}
	})
}
