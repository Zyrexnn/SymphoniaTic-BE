package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/services"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type EventController struct {
	repo          repositories.EventRepository
	broadcastRepo repositories.BroadcastRepository
}

func NewEventController(repo repositories.EventRepository) *EventController {
	return &EventController{repo: repo}
}

func (c *EventController) SetBroadcastRepo(b repositories.BroadcastRepository) {
	c.broadcastRepo = b
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

// GetEventSeats godoc
// @Summary Mengambil daftar nomor kursi dan status keterisiannya
// @Description Mengambil seluruh kursi fisik pertunjukan beserta status AVAILABLE atau BOOKED secara real-time.
// @Tags Public - Events
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.APIResponse{data=[]models.SeatItem} "Berhasil mengambil daftar kursi konser"
// @Failure 404 {object} models.APIResponse "Konser tidak ditemukan"
// @Router /events/{id}/seats [get]
func (c *EventController) GetEventSeats(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	seats, err := c.repo.GetSeatsByEventID(ctx.Context(), eventID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return utils.ResponseNotFound(ctx, "Konser tidak ditemukan")
		}
		return utils.ResponseInternalError(ctx, "Gagal mengambil daftar kursi konser", err)
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil daftar kursi konser", seats)
}

func validateEventSchedule(date, openGate, eventTime string, rundown []models.RundownItem, salesEndAt *time.Time) string {
	if strings.TrimSpace(openGate) == "" {
		return "Waktu Open Gate wajib diisi"
	}

	if valid, isBefore := utils.IsTimeBefore(openGate, eventTime); valid && !isBefore {
		return "Waktu Open Gate harus lebih awal dari waktu mulai pertunjukan konser"
	}

	for _, item := range rundown {
		if strings.TrimSpace(item.Time) == "" || strings.TrimSpace(item.Activity) == "" {
			return "Setiap item rundown wajib memiliki waktu dan aktivitas"
		}
	}

	if salesEndAt != nil && !salesEndAt.IsZero() {
		if concertDT, err := utils.ParseConcertDateTime(date, eventTime); err == nil && !concertDT.IsZero() {
			if salesEndAt.After(concertDT) {
				return "Batas waktu penjualan tiket tidak boleh melebihi waktu mulai konser"
			}
		}
	}

	return ""
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

	if errMsg := validateEventSchedule(createEventReq.Date, createEventReq.OpenGate, createEventReq.Time, createEventReq.Rundown, createEventReq.SalesEndAt); errMsg != "" {
		return utils.ResponseBadRequest(ctx, errMsg)
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

	if errMsg := validateEventSchedule(updateEventReq.Date, updateEventReq.OpenGate, updateEventReq.Time, updateEventReq.Rundown, updateEventReq.SalesEndAt); errMsg != "" {
		return utils.ResponseBadRequest(ctx, errMsg)
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
		if errors.Is(err, repositories.ErrEventHasOrders) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Konser tidak dapat dihapus karena sudah memiliki transaksi tiket aktif. Gunakan fitur Tutup Order.",
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
		if errors.Is(err, repositories.ErrEventCancelled) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Konser yang sudah dibatalkan tidak dapat dibuka kembali",
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

func (c *EventController) CancelEvent(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	var req models.CancelEventRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data pembatalan konser tidak valid",
			Error:   err.Error(),
		})
	}

	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Alasan pembatalan konser wajib diisi",
		})
	}

	updatedEvent, err := c.repo.CancelEvent(ctx.Context(), eventID, req.Reason)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Konser tidak ditemukan",
			})
		}
		if errors.Is(err, repositories.ErrEventAlreadyCancelled) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Konser ini sudah berstatus dibatalkan sebelumnya",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membatalkan konser",
			Error:   err.Error(),
		})
	}

	if req.NotifyAttendees && c.broadcastRepo != nil && updatedEvent != nil {
		createdBy, _ := ctx.Locals("admin_username").(string)
		if strings.TrimSpace(createdBy) == "" {
			createdBy = "admin"
		}
		c.dispatchEventLifecycleBroadcast(ctx.Context(), updatedEvent, "EVENT_CANCELLED",
			fmt.Sprintf("Pemberitahuan Pembatalan Konser: %s", updatedEvent.Title),
			fmt.Sprintf("Konser %s Dibatalkan", updatedEvent.Title),
			fmt.Sprintf("Halo {{name}},\n\nDengan berat hati kami informasikan bahwa konser \"%s\" (%s) telah dibatalkan.\n\nAlasan pembatalan:\n%s\n\nAnda berhak mengajukan refund penuh untuk pesanan {{orderCode}} tanpa batas waktu 24 jam melalui halaman Refund kami.", updatedEvent.Title, updatedEvent.Date, req.Reason),
			"Ajukan Refund Tiket", "/refund", createdBy)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Konser berhasil dibatalkan dan penjualan tiket telah ditutup",
		Data:    updatedEvent,
	})
}

func (c *EventController) RescheduleEvent(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	var req models.RescheduleEventRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data pengunduran jadwal konser tidak valid",
			Error:   err.Error(),
		})
	}

	req.NewDate = strings.TrimSpace(req.NewDate)
	req.NewTime = strings.TrimSpace(req.NewTime)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.NewDate == "" || req.NewTime == "" || req.Reason == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tanggal baru, waktu baru, dan alasan pengunduran jadwal wajib diisi",
		})
	}

	updatedEvent, err := c.repo.RescheduleEvent(ctx.Context(), eventID, &req)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Konser tidak ditemukan",
			})
		}
		if errors.Is(err, repositories.ErrEventCancelled) {
			return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
				Success: false,
				Message: "Konser yang sudah dibatalkan tidak dapat dijadwalkan ulang",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menjadwalkan ulang konser",
			Error:   err.Error(),
		})
	}

	if req.NotifyAttendees && c.broadcastRepo != nil && updatedEvent != nil {
		createdBy, _ := ctx.Locals("admin_username").(string)
		if strings.TrimSpace(createdBy) == "" {
			createdBy = "admin"
		}
		c.dispatchEventLifecycleBroadcast(ctx.Context(), updatedEvent, "EVENT_RESCHEDULED",
			fmt.Sprintf("Perubahan Jadwal Konser: %s", updatedEvent.Title),
			fmt.Sprintf("Jadwal Baru Konser %s", updatedEvent.Title),
			fmt.Sprintf("Halo {{name}},\n\nJadwal konser \"%s\" mengalami penyesuaian:\n- Jadwal Baru: %s pukul %s\n- Alasan: %s\n\nTiket dengan kode {{orderCode}} tetap berlaku untuk jadwal baru. Apabila berhalangan hadir, Anda dapat mengajukan refund melalui halaman Refund kami.", updatedEvent.Title, updatedEvent.Date, updatedEvent.Time, req.Reason),
			"Lihat Tiket / Ajukan Refund", "/refund", createdBy)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Jadwal konser berhasil diperbarui",
		Data:    updatedEvent,
	})
}

func (c *EventController) dispatchEventLifecycleBroadcast(ctx context.Context, evt *models.EventItem, category, subject, headline, bodyText, ctaLabel, ctaURL, createdBy string) {
	recipients, err := c.broadcastRepo.ResolveRecipients(ctx, "EVENT_ATTENDEES", evt.ID)
	if err != nil || len(recipients) == 0 {
		return
	}

	bcastID := "bcast-" + uuid.NewString()[:8]
	req := models.CreateBroadcastRequest{
		TargetScope:   "EVENT_ATTENDEES",
		TargetEventID: evt.ID,
		Category:      category,
		Subject:       subject,
		Headline:      headline,
		BodyText:      bodyText,
		CTALabel:      ctaLabel,
		CTAURL:        ctaURL,
	}

	record := &models.EmailBroadcastRecord{
		ID:              bcastID,
		TargetScope:     req.TargetScope,
		TargetEventID:   evt.ID,
		TargetEventName: evt.Title,
		Category:        category,
		Subject:         subject,
		Headline:        headline,
		BodyText:        bodyText,
		CTALabel:        ctaLabel,
		CTAURL:          ctaURL,
		TotalRecipients: len(recipients),
		Status:          "PROCESSING",
		CreatedBy:       createdBy,
		CreatedAt:       time.Now(),
	}

	if err := c.broadcastRepo.CreateBroadcast(ctx, record); err != nil {
		return
	}

	services.SendBroadcastBatchAsync(req, evt.Title, recipients, func(sent, failed int) {
		finalStatus := "COMPLETED"
		if sent == 0 && failed > 0 {
			finalStatus = "FAILED"
		} else if failed > 0 {
			finalStatus = "PARTIAL_FAILED"
		}
		_ = c.broadcastRepo.UpdateBroadcastProgress(context.Background(), bcastID, sent, failed, finalStatus)
	})
}
