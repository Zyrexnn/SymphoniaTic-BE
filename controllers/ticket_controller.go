package controllers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

type TicketController struct {
	repo repositories.OrderRepository
}

func NewTicketController(repo repositories.OrderRepository) *TicketController {
	return &TicketController{repo: repo}
}

// LookupTicketByCode godoc
// @Summary Lookup tiket berdasarkan kode pesanan
// @Description Mencari detail tiket E-Ticket berdasarkan order code (tanpa login).
// @Tags Public - Tickets
// @Accept json
// @Produce json
// @Param code query string true "Order Code (misal: SYM-XXXXXX)"
// @Success 200 {object} models.APIResponse{data=models.OrderRecord} "Tiket ditemukan"
// @Failure 400 {object} models.APIResponse "Parameter code wajib diisi"
// @Failure 404 {object} models.APIResponse "Tiket tidak ditemukan"
// @Router /tickets/lookup [get]
func (c *TicketController) LookupTicketByCode(ctx *fiber.Ctx) error {
	ticketOrderCode := ctx.Query("code")
	if ticketOrderCode == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Query parameter 'code' wajib diisi",
		})
	}

	foundOrder, err := c.repo.LookupTicketByCode(ctx.Context(), ticketOrderCode)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kode pesanan tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal melakukan pencarian tiket",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Tiket ditemukan",
		Data:    foundOrder,
	})
}

// CheckInTicket godoc
// @Summary Check-in tiket konser (Admin / Petugas)
// @Description Memvalidasi dan mengubah status tiket menjadi CHECKED_IN secara atomik.
// @Tags Admin - Tickets
// @Accept json
// @Produce json
// @Param payload body models.CheckInRequest true "Ticket Check-In Payload"
// @Success 200 {object} models.APIResponse{data=models.OrderRecord} "Check-in berhasil"
// @Failure 400 {object} models.APIResponse "Tiket sudah digunakan atau status tidak valid"
// @Failure 404 {object} models.APIResponse "Tiket tidak ditemukan"
// @Router /admin/tickets/checkin [post]
func (c *TicketController) CheckInTicket(ctx *fiber.Ctx) error {
	var req models.CheckInRequest
	_ = ctx.BodyParser(&req)

	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = strings.TrimSpace(req.OrderCode)
	}
	if code == "" {
		code = strings.TrimSpace(ctx.Query("code"))
	}
	if code == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode tiket atau order code wajib diisi",
		})
	}

	orderRecord, err := c.repo.CheckInTicket(ctx.Context(), code, req.BypassOpenGate)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kode pesanan tidak ditemukan",
			})
		}
		if errors.Is(err, repositories.ErrTicketAlreadyUsed) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Tiket ini sudah pernah digunakan untuk check-in sebelumnya",
			})
		}
		if errors.Is(err, repositories.ErrTicketInvalid) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Tiket tidak valid karena telah dibatalkan atau di-refund",
			})
		}
		if errors.Is(err, repositories.ErrTicketNotIssued) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Status tiket tidak valid untuk check-in",
			})
		}
		if errors.Is(err, repositories.ErrEventClosed) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Konser telah ditutup oleh panitia dan tiket tidak dapat digunakan untuk check-in",
			})
		}
		if errors.Is(err, repositories.ErrEventNotStarted) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Gerbang masuk (Open Gate) belum dibuka untuk konser ini",
			})
		}
		if errors.Is(err, repositories.ErrEventExpired) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Tiket tidak dapat digunakan karena acara konser telah selesai",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memproses check-in tiket",
			Error:   err.Error(),
		})
	}

	msg := "Check-in berhasil. Selamat menikmati pertunjukan!"
	if orderRecord != nil && strings.ToUpper(orderRecord.Status) != "CHECKED_IN" && orderRecord.CheckedInSeats != "" {
		checkedCount := len(models.ParseCommaList(orderRecord.CheckedInSeats))
		totalCount := orderRecord.Quantity
		if totalCount < checkedCount {
			totalCount = checkedCount
		}
		msg = fmt.Sprintf("Check-in kursi berhasil (%d/%d tiket telah masuk)", checkedCount, totalCount)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: msg,
		Data:    orderRecord,
	})
}

// GetAttendanceSummary godoc
// @Summary Ringkasan kehadiran penonton di gate (Admin / Petugas)
// @Description Menghitung jumlah tiket masuk (CHECKED_IN) dibanding total tiket aktif yang diterbitkan.
// @Tags Admin - Tickets
// @Accept json
// @Produce json
// @Param event_id query string false "Filter berdasarkan ID Konser (opsional)"
// @Success 200 {object} models.APIResponse{data=models.GateAttendanceSummary} "Ringkasan kehadiran berhasil dimuat"
// @Failure 500 {object} models.APIResponse "Gagal menghitung ringkasan kehadiran"
// @Router /admin/tickets/attendance-summary [get]
func (c *TicketController) GetAttendanceSummary(ctx *fiber.Ctx) error {
	eventID := strings.TrimSpace(ctx.Query("event_id"))
	summary, err := c.repo.GetAttendanceSummary(ctx.Context(), eventID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memuat ringkasan kehadiran gate",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Ringkasan kehadiran gate berhasil dimuat",
		Data:    summary,
	})
}
