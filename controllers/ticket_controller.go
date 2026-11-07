package controllers

import (
	"errors"

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
