package controllers

import (
	"errors"
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

	orderRecord, err := c.repo.CheckInTicket(ctx.Context(), code)
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
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memproses check-in tiket",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Check-in berhasil. Selamat menikmati pertunjukan!",
		Data:    orderRecord,
	})
}
