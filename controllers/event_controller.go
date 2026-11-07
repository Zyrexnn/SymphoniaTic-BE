package controllers

import (
	"errors"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

type EventController struct {
	repo repositories.EventRepository
}

func NewEventController(repo repositories.EventRepository) *EventController {
	return &EventController{repo: repo}
}

// GetEvents godoc
// @Summary Mengambil daftar seluruh konser
// @Description Mengambil semua event konser yang terdaftar beserta kategori tiketnya.
// @Tags Public - Events
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.EventItem} "Berhasil mengambil data konser"
// @Failure 500 {object} models.APIResponse "Gagal mengambil data konser"
// @Router /events [get]
func (c *EventController) GetEvents(ctx *fiber.Ctx) error {
	events, err := c.repo.GetAllEvents(ctx.Context())
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil data konser", err)
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil data konser", events)
}

// GetEventByID godoc
// @Summary Mengambil detail konser berdasarkan ID
// @Description Mengambil detail spesifik dari satu konser berdasarkan ID.
// @Tags Public - Events
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.APIResponse{data=models.EventItem} "Berhasil mengambil detail konser"
// @Failure 404 {object} models.APIResponse "Konser tidak ditemukan"
// @Failure 500 {object} models.APIResponse "Gagal mengambil detail konser"
// @Router /events/{id} [get]
func (c *EventController) GetEventByID(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	event, err := c.repo.GetEventByID(ctx.Context(), eventID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return utils.ResponseNotFound(ctx, "Konser tidak ditemukan")
		}
		return utils.ResponseInternalError(ctx, "Gagal mengambil detail konser", err)
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil detail konser", event)
}

// CreateEvent godoc
// @Summary Membuat event/konser baru (Admin)
// @Description Menambahkan event konser baru ke dalam sistem beserta kategori tiket awal.
// @Tags Admin - Events Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body models.CreateEventRequest true "Create Event Payload"
// @Success 201 {object} models.APIResponse{data=models.EventItem} "Berhasil menambahkan konser baru"
// @Failure 400 {object} models.APIResponse "Judul, Artis, Venue, Tanggal wajib diisi"
// @Router /admin/events [post]
func (c *EventController) CreateEvent(ctx *fiber.Ctx) error {
	var createEventReq models.CreateEventRequest
	if err := ctx.BodyParser(&createEventReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if createEventReq.Title == "" || createEventReq.Artist == "" || createEventReq.Venue == "" || createEventReq.Date == "" || createEventReq.Time == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Judul, Artist, Venue, Tanggal, dan Waktu wajib diisi",
		})
	}

	if createEventReq.Category == "" {
		createEventReq.Category = "SIMFONI UTAMA"
	}
	if createEventReq.CategoryBadgeColor == "" {
		createEventReq.CategoryBadgeColor = "bg-blue-900/80 text-blue-200 border-blue-500/40"
	}

	createdEvent, err := c.repo.CreateEvent(ctx.Context(), &createEventReq)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menyimpan event konser ke database",
			Error:   err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Event konser berhasil ditambahkan",
		Data:    createdEvent,
	})
}

// UpdateEvent godoc
// @Summary Memperbarui data konser (Admin)
// @Description Memperbarui rincian metadata konser musik klasik berdasarkan ID konser.
// @Tags Admin - Events Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param payload body models.UpdateEventRequest true "Update Event Payload"
// @Success 200 {object} models.APIResponse "Event konser berhasil diperbarui"
// @Failure 400 {object} models.APIResponse "Format request tidak valid"
// @Failure 404 {object} models.APIResponse "Event konser tidak ditemukan"
// @Router /admin/events/{id} [put]
func (c *EventController) UpdateEvent(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	var updateEventReq models.UpdateEventRequest
	if err := ctx.BodyParser(&updateEventReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	err := c.repo.UpdateEvent(ctx.Context(), eventID, &updateEventReq)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Event konser tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui event konser",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Event konser berhasil diperbarui",
	})
}

// DeleteEvent godoc
// @Summary Menghapus konser dan kategori tiket terkait (Admin)
// @Description Menghapus acara konser dari sistem beserta seluruh kategori tiket yang berada di bawahnya.
// @Tags Admin - Events Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.APIResponse "Event konser dan kategori tiket terkait berhasil dihapus"
// @Failure 404 {object} models.APIResponse "Event konser tidak ditemukan"
// @Router /admin/events/{id} [delete]
func (c *EventController) DeleteEvent(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	err := c.repo.DeleteEvent(ctx.Context(), eventID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Event konser tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menghapus event konser",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Event konser dan kategori tiket terkait berhasil dihapus",
	})
}

// ToggleEventClose godoc
// @Summary Membuka atau menutup penjualan tiket konser (Admin)
// @Description Mengubah status penjualan tiket konser antara terbuka atau ditutup secara instan.
// @Tags Admin - Events Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.APIResponse "Status penutupan konser berhasil diperbarui"
// @Failure 404 {object} models.APIResponse "Konser tidak ditemukan"
// @Router /admin/events/{id}/toggle-close [patch]
func (c *EventController) ToggleEventClose(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	newClosedState, err := c.repo.ToggleEventClose(ctx.Context(), eventID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Konser tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengecek status konser",
			Error:   err.Error(),
		})
	}

	msg := "Penjualan tiket konser berhasil ditutup"
	if !newClosedState {
		msg = "Penjualan tiket konser berhasil dibuka kembali"
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: msg,
		Data: fiber.Map{
			"eventId":  eventID,
			"isClosed": newClosedState,
		},
	})
}
