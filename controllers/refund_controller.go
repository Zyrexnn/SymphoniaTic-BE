package controllers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/services"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

type RefundController struct {
	repo      repositories.RefundRepository
	orderRepo repositories.OrderRepository
}

func NewRefundController(repo repositories.RefundRepository) *RefundController {
	var orderRepo repositories.OrderRepository
	if database.DB != nil {
		orderRepo = repositories.NewOrderRepository(database.DB)
	}
	return &RefundController{
		repo:      repo,
		orderRepo: orderRepo,
	}
}

func NewRefundControllerWithOrderRepo(repo repositories.RefundRepository, orderRepo repositories.OrderRepository) *RefundController {
	return &RefundController{
		repo:      repo,
		orderRepo: orderRepo,
	}
}

// RequestRefundOTP godoc
// @Summary Meminta kode OTP untuk pengajuan refund
// @Description Meminta kode OTP yang akan dikirimkan ke email pemilik tiket untuk validasi refund.
// @Tags Refunds
// @Accept json
// @Produce json
// @Param payload body models.RequestRefundOTPInput true "Request Refund OTP Payload"
// @Success 200 {object} models.APIResponse "Kode OTP verifikasi refund telah dikirim"
// @Failure 400 {object} models.APIResponse "Data request tidak valid / Tiket tidak memenuhi syarat refund"
// @Failure 404 {object} models.APIResponse "Kode pesanan tidak ditemukan"
// @Router /refunds/request-otp [post]
func (c *RefundController) RequestRefundOTP(ctx *fiber.Ctx) error {
	var requestOTPInput models.RequestRefundOTPInput
	if err := ctx.BodyParser(&requestOTPInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
			Error:   err.Error(),
		})
	}

	orderCode := strings.TrimSpace(requestOTPInput.OrderCode)
	userEmail := strings.ToLower(strings.TrimSpace(requestOTPInput.UserEmail))

	if orderCode == "" || userEmail == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode pesanan dan email wajib diisi",
		})
	}

	if c.orderRepo == nil && database.DB != nil {
		c.orderRepo = repositories.NewOrderRepository(database.DB)
	}
	if c.orderRepo == nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Layanan verifikasi pesanan tidak tersedia",
		})
	}

	targetOrder, err := c.orderRepo.GetOrderByCode(ctx.Context(), orderCode)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kode pesanan tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memverifikasi pesanan",
			Error:   err.Error(),
		})
	}

	if !strings.EqualFold(targetOrder.UserEmail, userEmail) {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Email tidak cocok dengan pemegang tiket ini",
		})
	}

	if targetOrder.Status == "CHECKED_IN" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tiket sudah di-scan (Check In) pada venue dan tidak dapat dikembalikan / refund",
		})
	}
	if targetOrder.Status == "REFUND_REQUESTED" || targetOrder.Status == "REFUNDED" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Pengajuan refund untuk tiket ini sedang diproses atau sudah selesai",
		})
	}
	if targetOrder.Status == "CANCELLED" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Pesanan ini sudah dibatalkan sebelumnya",
		})
	}

	otpCode, err := utils.GenerateCryptoOTP()
	if err != nil {
		otpCode = fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	expiresAt := time.Now().Add(10 * time.Minute)

	err = c.repo.RequestRefundOTP(ctx.Context(), targetOrder.OrderCode, targetOrder.UserEmail, otpCode, expiresAt, targetOrder.TotalPrice, targetOrder.ID)
	if err != nil {
		if errors.Is(err, repositories.ErrOTPCooldown) {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(models.APIResponse{
				Success: false,
				Message: "Mohon tunggu 60 detik sebelum meminta kode OTP refund baru",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat kode verifikasi OTP",
			Error:   err.Error(),
		})
	}

	services.SendRefundOTPEmail(targetOrder.UserEmail, targetOrder.OrderCode, otpCode)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Kode OTP verifikasi refund telah dikirim ke email " + targetOrder.UserEmail,
		Data: fiber.Map{
			"orderCode": targetOrder.OrderCode,
			"userEmail": targetOrder.UserEmail,
			"expiresIn": "10 Menit",
		},
	})
}

// SubmitRefund godoc
// @Summary Mengajukan formulir permohonan pengembalian dana (refund)
// @Description Mengirimkan detail rekening bank dan alasan refund setelah memverifikasi kode OTP.
// @Tags Refunds
// @Accept json
// @Produce json
// @Param payload body models.SubmitRefundInput true "Submit Refund Payload"
// @Success 200 {object} models.APIResponse "Permohonan pengembalian dana berhasil diajukan"
// @Failure 400 {object} models.APIResponse "Data tidak lengkap atau OTP tidak valid"
// @Failure 404 {object} models.APIResponse "Pengajuan refund tidak ditemukan"
// @Router /refunds/submit [post]
func (c *RefundController) SubmitRefund(ctx *fiber.Ctx) error {
	var submitRefundInput models.SubmitRefundInput
	if err := ctx.BodyParser(&submitRefundInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
			Error:   err.Error(),
		})
	}

	submitRefundInput.OrderCode = strings.TrimSpace(submitRefundInput.OrderCode)
	submitRefundInput.UserEmail = strings.ToLower(strings.TrimSpace(submitRefundInput.UserEmail))
	submitRefundInput.OTPCode = strings.TrimSpace(submitRefundInput.OTPCode)
	submitRefundInput.BankName = strings.TrimSpace(submitRefundInput.BankName)
	submitRefundInput.AccountNumber = strings.TrimSpace(submitRefundInput.AccountNumber)
	submitRefundInput.AccountHolder = strings.TrimSpace(submitRefundInput.AccountHolder)
	submitRefundInput.Reason = strings.TrimSpace(submitRefundInput.Reason)

	if submitRefundInput.OrderCode == "" || submitRefundInput.UserEmail == "" || submitRefundInput.OTPCode == "" ||
		submitRefundInput.BankName == "" || submitRefundInput.AccountNumber == "" || submitRefundInput.AccountHolder == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Seluruh kolom formulir dan OTP wajib diisi",
		})
	}

	refundRecord, err := c.repo.SubmitRefund(ctx.Context(), &submitRefundInput)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Pengajuan refund untuk kode pesanan ini tidak ditemukan",
			})
		}
		if errors.Is(err, repositories.ErrUnauthorized) {
			return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
				Success: false,
				Message: "Email tidak cocok dengan permohonan refund",
			})
		}
		if errors.Is(err, repositories.ErrOTPMaxExceed) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Batas percobaan verifikasi OTP telah terlampaui. Silakan minta kode OTP baru",
			})
		}
		var errWrongCode *repositories.ErrOTPWrongCode
		if errors.As(err, &errWrongCode) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: fmt.Sprintf("Kode OTP salah. Sisa percobaan: %d", errWrongCode.RemainingAttempts),
			})
		}
		if errors.Is(err, repositories.ErrOTPInvalid) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Kode OTP salah atau sudah kedaluwarsa",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengajukan permohonan refund",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Pengajuan refund berhasil dikirim. Tim Finance akan meninjau dan memproses pengembalian dana Anda.",
		Data: fiber.Map{
			"orderCode":     refundRecord.OrderCode,
			"status":        "PENDING",
			"bankName":      submitRefundInput.BankName,
			"accountHolder": submitRefundInput.AccountHolder,
		},
	})
}

// GetRefundStatus godoc
// @Summary Memeriksa status pengajuan refund tiket
// @Description Mengecek status terkini (PENDING, APPROVED, REJECTED, COMPLETED) dari pengajuan refund.
// @Tags Refunds
// @Accept json
// @Produce json
// @Param payload body models.CheckRefundStatusRequest true "Check Refund Status Payload"
// @Success 200 {object} models.APIResponse "Berhasil mengambil status refund"
// @Failure 404 {object} models.APIResponse "Data pengajuan refund tidak ditemukan"
// @Router /refunds/status [post]
func (c *RefundController) GetRefundStatus(ctx *fiber.Ctx) error {
	var checkStatusInput models.CheckRefundStatusRequest
	if err := ctx.BodyParser(&checkStatusInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
		})
	}

	orderCode := strings.TrimSpace(checkStatusInput.OrderCode)
	userEmail := strings.ToLower(strings.TrimSpace(checkStatusInput.UserEmail))

	if orderCode == "" || userEmail == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode pesanan dan email wajib diisi",
		})
	}

	refundRecord, orderStatus, err := c.repo.GetRefundStatus(ctx.Context(), orderCode, userEmail)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Data pengajuan refund tidak ditemukan untuk kode dan email ini",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil status refund",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil mengambil status refund",
		Data: fiber.Map{
			"orderStatus":  orderStatus,
			"refundDetail": refundRecord,
		},
	})
}

// AdminGetAllRefunds godoc
// @Summary Mengambil semua permohonan refund (Admin)
// @Description Mengambil seluruh permohonan refund tiket yang diajukan oleh pengguna.
// @Tags Admin - Refund Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.RefundRequestRecord} "Berhasil mengambil daftar permohonan refund"
// @Router /admin/refunds [get]
func (c *RefundController) AdminGetAllRefunds(ctx *fiber.Ctx) error {
	refundList, err := c.repo.GetAllRefunds(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil daftar permohonan refund",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil mengambil daftar permohonan refund",
		Data:    refundList,
	})
}

// AdminUpdateRefundStatus godoc
// @Summary Menyetujui atau menolak permohonan refund (Admin)
// @Description Memperbarui status permohonan refund menjadi APPROVED, REJECTED, atau COMPLETED beserta catatan admin.
// @Tags Admin - Refund Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Refund ID"
// @Param payload body models.UpdateRefundStatusRequest true "Update Refund Status Payload"
// @Success 200 {object} models.APIResponse "Status pengajuan refund berhasil diubah"
// @Failure 400 {object} models.APIResponse "Status tidak valid"
// @Failure 404 {object} models.APIResponse "Data refund tidak ditemukan"
// @Router /admin/refunds/{id}/status [patch]
func (c *RefundController) AdminUpdateRefundStatus(ctx *fiber.Ctx) error {
	refundID := ctx.Params("id")
	var updateRefundStatusReq models.UpdateRefundStatusRequest
	if err := ctx.BodyParser(&updateRefundStatusReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
		})
	}

	newStatus := strings.ToUpper(strings.TrimSpace(updateRefundStatusReq.Status))
	if newStatus != "APPROVED" && newStatus != "REJECTED" && newStatus != "COMPLETED" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Status harus APPROVED, REJECTED, atau COMPLETED",
		})
	}

	updated, err := c.repo.UpdateRefundStatus(ctx.Context(), refundID, newStatus, updateRefundStatusReq.AdminNote)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Data refund tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status refund",
			Error:   err.Error(),
		})
	}

	services.SendRefundStatusNotificationEmail(updated.UserEmail, updated.OrderCode, updated.Status, updateRefundStatusReq.AdminNote, updated.RefundAmount)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Status pengajuan refund berhasil diubah menjadi %s", newStatus),
	})
}
