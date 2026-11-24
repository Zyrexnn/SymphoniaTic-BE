package controllers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type BroadcastController struct {
	repo      repositories.BroadcastRepository
	eventRepo repositories.EventRepository
}

func NewBroadcastController(repo repositories.BroadcastRepository, eventRepo repositories.EventRepository) *BroadcastController {
	return &BroadcastController{
		repo:      repo,
		eventRepo: eventRepo,
	}
}

func (c *BroadcastController) GetAllBroadcasts(ctx *fiber.Ctx) error {
	list, err := c.repo.GetAllBroadcasts(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memuat riwayat broadcast email",
			Error:   err.Error(),
		})
	}
	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil memuat riwayat broadcast email",
		Data:    list,
	})
}

func (c *BroadcastController) CreateBroadcast(ctx *fiber.Ctx) error {
	var req models.CreateBroadcastRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format payload broadcast email tidak valid",
			Error:   err.Error(),
		})
	}

	req.TargetScope = strings.ToUpper(strings.TrimSpace(req.TargetScope))
	req.TargetEventID = strings.TrimSpace(req.TargetEventID)
	req.TargetCategoryID = strings.TrimSpace(req.TargetCategoryID)
	req.Category = strings.ToUpper(strings.TrimSpace(req.Category))
	req.Subject = strings.TrimSpace(req.Subject)
	req.Headline = strings.TrimSpace(req.Headline)
	req.BodyText = strings.TrimSpace(req.BodyText)
	req.PromoCode = strings.ToUpper(strings.TrimSpace(req.PromoCode))
	req.CTALabel = strings.TrimSpace(req.CTALabel)
	req.CTAURL = strings.TrimSpace(req.CTAURL)

	switch req.TargetScope {
	case "ALL_USERS", "EVENT_ATTENDEES", "CATEGORY_ATTENDEES", "ALL_CUSTOMERS":
	default:
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Target penerima tidak valid. Pilih ALL_USERS, EVENT_ATTENDEES, CATEGORY_ATTENDEES, atau ALL_CUSTOMERS",
		})
	}

	if (req.TargetScope == "EVENT_ATTENDEES" || req.TargetScope == "CATEGORY_ATTENDEES") && req.TargetEventID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Event konser wajib dipilih untuk target pemegang tiket konser",
		})
	}
	if req.TargetScope == "CATEGORY_ATTENDEES" && req.TargetCategoryID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kategori tiket wajib dipilih untuk target pemegang tiket kategori",
		})
	}

	if req.Category == "" {
		req.Category = "ANNOUNCEMENT"
	}
	switch req.Category {
	case "ANNOUNCEMENT", "PROMO", "EVENT_CANCELLED", "EVENT_RESCHEDULED":
	default:
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kategori broadcast tidak valid",
		})
	}

	if req.Subject == "" || req.Headline == "" || req.BodyText == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Subjek, judul utama, dan isi pesan email wajib diisi",
		})
	}

	var eventTitle string
	if req.TargetEventID != "" && c.eventRepo != nil {
		if evt, err := c.eventRepo.GetEventByID(ctx.Context(), req.TargetEventID); err == nil && evt != nil {
			eventTitle = evt.Title
		}
	}

	excludePromoOptOut := req.Category == "PROMO"
	recipients, err := c.repo.ResolveRecipientsFiltered(ctx.Context(), req.TargetScope, req.TargetEventID, req.TargetCategoryID, excludePromoOptOut)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengumpulkan daftar penerima email",
			Error:   err.Error(),
		})
	}

	if len(recipients) == 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tidak ada penerima yang cocok untuk target broadcast ini",
		})
	}

	createdBy, _ := ctx.Locals("admin_username").(string)
	if strings.TrimSpace(createdBy) == "" {
		createdBy = "admin"
	}

	record := &models.EmailBroadcastRecord{
		ID:               "bcast-" + uuid.NewString()[:8],
		TargetScope:      req.TargetScope,
		TargetEventID:    req.TargetEventID,
		TargetEventName:  eventTitle,
		TargetCategoryID: req.TargetCategoryID,
		Category:         req.Category,
		Subject:          req.Subject,
		Headline:         req.Headline,
		BodyText:         req.BodyText,
		CTALabel:         req.CTALabel,
		CTAURL:           req.CTAURL,
		PromoCode:        req.PromoCode,
		TotalRecipients:  len(recipients),
		Status:           "PROCESSING",
		CreatedBy:        createdBy,
		CreatedAt:        time.Now(),
	}

	if err := c.repo.CreateBroadcast(ctx.Context(), record); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mencatat log broadcast email",
			Error:   err.Error(),
		})
	}
	_ = c.repo.CreateBroadcastRecipients(ctx.Context(), record.ID, recipients)

	broadcastID := record.ID
	services.SendBroadcastBatchWithTrackerAsync(req, eventTitle, recipients,
		func(email, status, errMsg string) {
			_ = c.repo.UpdateRecipientStatus(context.Background(), broadcastID, email, status, errMsg)
		},
		func(sent, failed int) {
			finalStatus := "COMPLETED"
			if sent == 0 && failed > 0 {
				finalStatus = "FAILED"
			} else if failed > 0 {
				finalStatus = "PARTIAL_FAILED"
			}
			_ = c.repo.UpdateBroadcastProgress(context.Background(), broadcastID, sent, failed, finalStatus)
		},
	)

	return ctx.Status(fiber.StatusAccepted).JSON(models.APIResponse{
		Success: true,
		Message: "Broadcast email sedang diproses dan dikirim secara asinkron",
		Data:    record,
	})
}

func (c *BroadcastController) GetBroadcastRecipients(ctx *fiber.Ctx) error {
	broadcastID := strings.TrimSpace(ctx.Params("id"))
	if broadcastID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "ID broadcast wajib diisi",
		})
	}
	list, err := c.repo.GetBroadcastRecipients(ctx.Context(), broadcastID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memuat daftar penerima broadcast",
			Error:   err.Error(),
		})
	}
	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil memuat daftar penerima broadcast",
		Data:    list,
	})
}

func (c *BroadcastController) RetryFailedBroadcast(ctx *fiber.Ctx) error {
	broadcastID := strings.TrimSpace(ctx.Params("id"))
	if broadcastID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "ID broadcast wajib diisi",
		})
	}

	bcast, err := c.repo.GetBroadcastByID(ctx.Context(), broadcastID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Riwayat broadcast tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memeriksa data broadcast",
			Error:   err.Error(),
		})
	}

	failedRecipients, err := c.repo.GetFailedRecipients(ctx.Context(), broadcastID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memuat daftar penerima yang gagal",
			Error:   err.Error(),
		})
	}
	if len(failedRecipients) == 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tidak ada penerima gagal yang perlu dikirim ulang",
		})
	}

	_ = c.repo.UpdateBroadcastProgress(ctx.Context(), broadcastID, bcast.SentCount, len(failedRecipients), "PROCESSING")

	req := models.CreateBroadcastRequest{
		TargetScope:      bcast.TargetScope,
		TargetEventID:    bcast.TargetEventID,
		TargetCategoryID: bcast.TargetCategoryID,
		Category:         bcast.Category,
		Subject:          bcast.Subject,
		Headline:         bcast.Headline,
		BodyText:         bcast.BodyText,
		CTALabel:         bcast.CTALabel,
		CTAURL:           bcast.CTAURL,
		PromoCode:        bcast.PromoCode,
	}

	prevSent := bcast.SentCount
	services.SendBroadcastBatchWithTrackerAsync(req, bcast.TargetEventName, failedRecipients,
		func(email, status, errMsg string) {
			_ = c.repo.UpdateRecipientStatus(context.Background(), broadcastID, email, status, errMsg)
		},
		func(sent, failed int) {
			totalSent := prevSent + sent
			finalStatus := "COMPLETED"
			if totalSent == 0 && failed > 0 {
				finalStatus = "FAILED"
			} else if failed > 0 {
				finalStatus = "PARTIAL_FAILED"
			}
			_ = c.repo.UpdateBroadcastProgress(context.Background(), broadcastID, totalSent, failed, finalStatus)
		},
	)

	return ctx.Status(fiber.StatusAccepted).JSON(models.APIResponse{
		Success: true,
		Message: "Pengiriman ulang untuk penerima yang gagal sedang diproses",
		Data: fiber.Map{
			"broadcastId": broadcastID,
			"retryCount":  len(failedRecipients),
		},
	})
}

func (c *BroadcastController) UnsubscribePromo(ctx *fiber.Ctx) error {
	token := strings.TrimSpace(ctx.Query("token"))
	if token == "" {
		var body struct {
			Token string `json:"token"`
		}
		_ = ctx.BodyParser(&body)
		token = strings.TrimSpace(body.Token)
	}
	if token == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Token berhenti berlangganan wajib diisi",
		})
	}

	if err := c.repo.UnsubscribeByToken(ctx.Context(), token); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Token berhenti berlangganan tidak valid atau sudah tidak berlaku",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memproses permintaan berhenti berlangganan",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Anda berhasil berhenti berlangganan email promo SymphoniaTic. Email terkait tiket dan informasi konser Anda tetap akan dikirimkan.",
	})
}
