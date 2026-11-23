package controllers

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

var promoCodePattern = regexp.MustCompile(`^[A-Z0-9_-]{3,32}$`)

type PromoController struct {
	repo repositories.PromoRepository
}

func NewPromoController(repo repositories.PromoRepository) *PromoController {
	return &PromoController{repo: repo}
}

func validateAndNormalizePromoInput(req *models.CreatePromoRequest) string {
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	req.Description = strings.TrimSpace(req.Description)
	req.DiscountType = strings.ToUpper(strings.TrimSpace(req.DiscountType))

	if !promoCodePattern.MatchString(req.Code) {
		return "Kode promo wajib 3-32 karakter (hanya huruf, angka, '-' atau '_')"
	}
	if req.DiscountType != "PERCENTAGE" && req.DiscountType != "FIXED" {
		return "Tipe diskon harus PERCENTAGE atau FIXED"
	}
	if req.DiscountVal <= 0 {
		return "Nilai diskon harus lebih dari 0"
	}
	if req.DiscountType == "PERCENTAGE" && req.DiscountVal > 100 {
		return "Persentase diskon tidak boleh melebihi 100%"
	}
	if req.MinOrderAmount < 0 || req.MaxDiscount < 0 {
		return "Nominal minimal transaksi dan batas potongan tidak boleh negatif"
	}
	if req.Quota < 1 {
		return "Kuota promo minimal 1 pemakaian"
	}
	if req.ValidFrom.IsZero() || req.ValidUntil.IsZero() || !req.ValidUntil.After(req.ValidFrom) {
		return "Tanggal berakhir promo harus lebih lambat dari tanggal mulai"
	}
	return ""
}

// ValidatePromo godoc
// @Summary Memvalidasi kode promo dan menghitung estimasi potongan harga
// @Description Memeriksa ketersediaan kuota promo, status aktif, masa berlaku, syarat anggota login, dan minimal transaksi.
// @Tags Public - Promos
// @Accept json
// @Produce json
// @Param payload body models.ValidatePromoRequest true "Validate Promo Request Payload"
// @Success 200 {object} models.APIResponse{data=models.ValidatePromoResponse} "Kode promo valid"
// @Failure 400 {object} models.APIResponse "Kode promo tidak valid atau tidak memenuhi syarat"
// @Router /promos/validate [post]
func (c *PromoController) ValidatePromo(ctx *fiber.Ctx) error {
	var req models.ValidatePromoRequest
	if err := ctx.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(ctx, "Format permintaan validasi promo tidak valid")
	}

	req.PromoCode = strings.ToUpper(strings.TrimSpace(req.PromoCode))
	if req.PromoCode == "" {
		return utils.ResponseBadRequest(ctx, "Kode promo wajib diisi")
	}
	if req.Subtotal <= 0 {
		return utils.ResponseBadRequest(ctx, "Subtotal pemesanan tidak valid")
	}

	var authenticatedUserID string
	rawAuthHeader := ctx.Get("Authorization")
	if rawAuthHeader != "" && strings.HasPrefix(rawAuthHeader, "Bearer ") {
		rawToken := strings.TrimPrefix(rawAuthHeader, "Bearer ")
		claims, err := utils.ValidateUserToken(rawToken)
		if err == nil && claims != nil {
			authenticatedUserID = claims.UserID
		}
	}

	promo, err := c.repo.GetPromoByCode(ctx.Context(), req.PromoCode)
	if err != nil {
		if errors.Is(err, repositories.ErrPromoInvalid) {
			return utils.ResponseBadRequest(ctx, repositories.ErrPromoInvalid.Error())
		}
		return utils.ResponseInternalError(ctx, "Gagal memeriksa kode promo", err)
	}
	if promo == nil {
		return utils.ResponseBadRequest(ctx, repositories.ErrPromoInvalid.Error())
	}

	if !promo.IsActive {
		return utils.ResponseBadRequest(ctx, repositories.ErrPromoInactive.Error())
	}

	if promo.IsMemberOnly && authenticatedUserID == "" {
		return utils.ResponseBadRequest(ctx, repositories.ErrPromoMemberOnly.Error())
	}

	now := time.Now()
	if now.Before(promo.ValidFrom) || now.After(promo.ValidUntil) {
		return utils.ResponseBadRequest(ctx, repositories.ErrPromoExpired.Error())
	}

	if promo.RemainingQuota <= 0 {
		return utils.ResponseBadRequest(ctx, repositories.ErrPromoQuotaExceeded.Error())
	}

	if req.Subtotal < promo.MinOrderAmount {
		return utils.ResponseBadRequest(ctx, fmt.Sprintf("Minimal transaksi Rp %.0f untuk menggunakan kode promo ini", promo.MinOrderAmount))
	}

	if authenticatedUserID != "" {
		alreadyUsed, err := c.repo.HasUserUsedPromo(ctx.Context(), promo.ID, authenticatedUserID)
		if err != nil {
			return utils.ResponseInternalError(ctx, "Gagal memverifikasi pemakaian kode promo", err)
		}
		if alreadyUsed {
			return utils.ResponseBadRequest(ctx, repositories.ErrPromoAlreadyUsed.Error())
		}
	}

	discount := promo.CalculateDiscount(req.Subtotal)
	finalTotal := math.Max(0, req.Subtotal-discount)

	return utils.ResponseOK(ctx, "Kode promo berhasil diterapkan", models.ValidatePromoResponse{
		Valid:          true,
		Code:           promo.Code,
		Description:    promo.Description,
		DiscountType:   promo.DiscountType,
		DiscountAmount: discount,
		FinalTotal:     finalTotal,
	})
}

// GetActivePromos godoc
// @Summary Mengambil daftar kode promo yang sedang aktif
// @Description Mengembalikan daftar kode promo aktif yang masih memiliki kuota dan berada dalam periode berlaku.
// @Tags Public - Promos
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.Promotion} "Berhasil mengambil daftar promo aktif"
// @Router /promos/active [get]
func (c *PromoController) GetActivePromos(ctx *fiber.Ctx) error {
	promos, err := c.repo.GetActivePromos(ctx.Context())
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil daftar promo aktif", err)
	}
	return utils.ResponseOK(ctx, "Berhasil mengambil daftar promo aktif", promos)
}

// AdminGetAllPromos godoc
// @Summary Mengambil seluruh data kode promo (Admin)
// @Description Mengambil semua kode promo termasuk yang nonaktif, kedaluwarsa, maupun habis kuota untuk dikelola di dashboard admin.
// @Tags Admin - Promo Management
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.Promotion} "Berhasil mengambil daftar kode promo"
// @Router /admin/promos [get]
func (c *PromoController) AdminGetAllPromos(ctx *fiber.Ctx) error {
	promos, err := c.repo.GetAllPromos(ctx.Context())
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil daftar kode promo", err)
	}
	return utils.ResponseOK(ctx, "Daftar kode promo berhasil diambil", promos)
}

// AdminCreatePromo godoc
// @Summary Membuat kode promo kustom baru (Admin)
// @Description Menambahkan kode promo kustom dengan tipe persentase atau nominal tetap, batas kuota, serta masa berlaku.
// @Tags Admin - Promo Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body models.CreatePromoRequest true "Create Promo Payload"
// @Success 201 {object} models.APIResponse{data=models.Promotion} "Kode promo berhasil dibuat"
// @Failure 400 {object} models.APIResponse "Input tidak valid atau kode sudah terdaftar"
// @Router /admin/promos [post]
func (c *PromoController) AdminCreatePromo(ctx *fiber.Ctx) error {
	var req models.CreatePromoRequest
	if err := ctx.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload kode promo tidak valid")
	}

	if errMsg := validateAndNormalizePromoInput(&req); errMsg != "" {
		return utils.ResponseBadRequest(ctx, errMsg)
	}

	created, err := c.repo.CreatePromo(ctx.Context(), &req)
	if err != nil {
		if errors.Is(err, repositories.ErrPromoDuplicateCode) {
			return utils.ResponseBadRequest(ctx, repositories.ErrPromoDuplicateCode.Error())
		}
		return utils.ResponseInternalError(ctx, "Gagal membuat kode promo", err)
	}

	return utils.ResponseCreated(ctx, "Kode promo berhasil dibuat", created)
}

// AdminUpdatePromo godoc
// @Summary Memperbarui rincian kode promo (Admin)
// @Description Memperbarui aturan diskon, kuota, dan masa berlaku kode promo yang sudah ada.
// @Tags Admin - Promo Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Promo ID"
// @Param payload body models.UpdatePromoRequest true "Update Promo Payload"
// @Success 200 {object} models.APIResponse{data=models.Promotion} "Kode promo berhasil diperbarui"
// @Failure 400 {object} models.APIResponse "Input tidak valid atau kode duplikat"
// @Failure 404 {object} models.APIResponse "Kode promo tidak ditemukan"
// @Router /admin/promos/{id} [put]
func (c *PromoController) AdminUpdatePromo(ctx *fiber.Ctx) error {
	promoID := strings.TrimSpace(ctx.Params("id"))
	if promoID == "" {
		return utils.ResponseBadRequest(ctx, "ID kode promo wajib diisi")
	}

	var req models.UpdatePromoRequest
	if err := ctx.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload pembaruan promo tidak valid")
	}

	if errMsg := validateAndNormalizePromoInput(&req); errMsg != "" {
		return utils.ResponseBadRequest(ctx, errMsg)
	}

	updated, err := c.repo.UpdatePromo(ctx.Context(), promoID, &req)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return utils.ResponseNotFound(ctx, "Kode promo tidak ditemukan")
		}
		if errors.Is(err, repositories.ErrPromoDuplicateCode) {
			return utils.ResponseBadRequest(ctx, repositories.ErrPromoDuplicateCode.Error())
		}
		return utils.ResponseInternalError(ctx, "Gagal memperbarui kode promo", err)
	}

	return utils.ResponseOK(ctx, "Kode promo berhasil diperbarui", updated)
}

// AdminTogglePromoActive godoc
// @Summary Mengaktifkan atau menonaktifkan kode promo (Admin)
// @Description Mengubah status aktif kode promo tanpa menghapus riwayat penggunaan.
// @Tags Admin - Promo Management
// @Security BearerAuth
// @Produce json
// @Param id path string true "Promo ID"
// @Success 200 {object} models.APIResponse "Status aktif promo diperbarui"
// @Failure 404 {object} models.APIResponse "Kode promo tidak ditemukan"
// @Router /admin/promos/{id}/toggle-active [patch]
func (c *PromoController) AdminTogglePromoActive(ctx *fiber.Ctx) error {
	promoID := strings.TrimSpace(ctx.Params("id"))
	if promoID == "" {
		return utils.ResponseBadRequest(ctx, "ID kode promo wajib diisi")
	}

	isActive, err := c.repo.TogglePromoActive(ctx.Context(), promoID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return utils.ResponseNotFound(ctx, "Kode promo tidak ditemukan")
		}
		return utils.ResponseInternalError(ctx, "Gagal mengubah status kode promo", err)
	}

	statusLabel := "dinonaktifkan"
	if isActive {
		statusLabel = "diaktifkan"
	}
	return utils.ResponseOK(ctx, fmt.Sprintf("Kode promo berhasil %s", statusLabel), fiber.Map{
		"isActive": isActive,
	})
}

// AdminDeletePromo godoc
// @Summary Menghapus kode promo (Admin)
// @Description Menghapus kode promo secara permanen dari basis data.
// @Tags Admin - Promo Management
// @Security BearerAuth
// @Produce json
// @Param id path string true "Promo ID"
// @Success 200 {object} models.APIResponse "Kode promo berhasil dihapus"
// @Failure 404 {object} models.APIResponse "Kode promo tidak ditemukan"
// @Router /admin/promos/{id} [delete]
func (c *PromoController) AdminDeletePromo(ctx *fiber.Ctx) error {
	promoID := strings.TrimSpace(ctx.Params("id"))
	if promoID == "" {
		return utils.ResponseBadRequest(ctx, "ID kode promo wajib diisi")
	}

	err := c.repo.DeletePromo(ctx.Context(), promoID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return utils.ResponseNotFound(ctx, "Kode promo tidak ditemukan")
		}
		return utils.ResponseInternalError(ctx, "Gagal menghapus kode promo", err)
	}

	return utils.ResponseOK(ctx, "Kode promo berhasil dihapus", nil)
}
