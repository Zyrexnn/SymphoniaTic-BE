package controllers

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

type SecurityController struct {
	secRepo   repositories.SecurityRepository
	auditRepo repositories.AdminUserRepository
	blockMgr  *middleware.IPBlocklistManager
}

func NewSecurityController(
	secRepo repositories.SecurityRepository,
	auditRepo repositories.AdminUserRepository,
	blockMgr *middleware.IPBlocklistManager,
) *SecurityController {
	return &SecurityController{
		secRepo:   secRepo,
		auditRepo: auditRepo,
		blockMgr:  blockMgr,
	}
}

func parseFlexibleDate(raw string, endOfDay bool) (*time.Time, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, clean); err == nil {
		return &t, nil
	}
	loc := time.FixedZone("WIB", 7*3600)
	if t, err := time.ParseInLocation("2006-01-02", clean, loc); err == nil {
		if endOfDay {
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		return &t, nil
	}
	return nil, fmt.Errorf("format tanggal tidak valid: %s", clean)
}

func (c *SecurityController) GetFilteredAuditLogs(ctx *fiber.Ctx) error {
	fromTime, err := parseFlexibleDate(ctx.Query("from"), false)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format tanggal awal (from) tidak valid. Gunakan YYYY-MM-DD.",
		})
	}
	toTime, err := parseFlexibleDate(ctx.Query("to"), true)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format tanggal akhir (to) tidak valid. Gunakan YYYY-MM-DD.",
		})
	}
	if fromTime != nil && toTime != nil && fromTime.After(*toTime) {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tanggal awal tidak boleh lebih besar dari tanggal akhir.",
		})
	}

	page := 1
	if p, err := strconv.Atoi(strings.TrimSpace(ctx.Query("page"))); err == nil && p > 0 {
		page = p
	}
	limit := 50
	if l, err := strconv.Atoi(strings.TrimSpace(ctx.Query("limit"))); err == nil && l > 0 {
		limit = l
	}

	filter := models.AuditLogFilter{
		Action:    ctx.Query("action"),
		Category:  ctx.Query("category"),
		Severity:  ctx.Query("severity"),
		ActorRole: ctx.Query("role"),
		IPAddress: ctx.Query("ip"),
		Search:    ctx.Query("search"),
		From:      fromTime,
		To:        toTime,
		Page:      page,
		Limit:     limit,
	}

	result, err := c.secRepo.GetAuditLogsFiltered(ctx.Context(), filter)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil data log audit keamanan", err)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Data log audit keamanan berhasil diambil",
		Data:    result,
	})
}

func (c *SecurityController) ListBlockedIPs(ctx *fiber.Ctx) error {
	activeOnly := !strings.EqualFold(strings.TrimSpace(ctx.Query("activeOnly")), "false")
	list, err := c.secRepo.ListBlockedIPs(ctx.Context(), activeOnly)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil daftar IP diblokir", err)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Daftar IP diblokir berhasil diambil",
		Data:    list,
	})
}

func (c *SecurityController) BlockIP(ctx *fiber.Ctx) error {
	var req models.BlockIPRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format permintaan tidak valid",
		})
	}

	cleanIP := strings.TrimSpace(req.IPAddress)
	if net.ParseIP(cleanIP) == nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format alamat IP tidak valid",
		})
	}

	callerIP := ctx.IP()
	if c.blockMgr != nil {
		callerIP = c.blockMgr.ResolveClientIP(ctx)
	}
	if cleanIP == callerIP {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tidak dapat memblokir alamat IP Anda sendiri untuk mencegah terkuncinya sesi Super Admin",
		})
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Diblokir secara manual oleh Super Admin"
	}

	var expiresAt *time.Time
	if req.DurationMinutes > 0 {
		t := time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute)
		expiresAt = &t
	}

	actorID, actorEmail, actorRole := resolveActorFromCtx(ctx)
	entry := &models.BlockedIP{
		IPAddress: cleanIP,
		Reason:    reason,
		BlockedBy: actorEmail,
		IsAuto:    false,
		ExpiresAt: expiresAt,
	}

	var saved *models.BlockedIP
	var err error
	if c.blockMgr != nil {
		saved, err = c.blockMgr.BlockIP(ctx.Context(), entry)
	} else {
		saved, err = c.secRepo.BlockIP(ctx.Context(), entry)
	}
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal memblokir alamat IP", err)
	}

	if c.auditRepo != nil {
		_ = c.auditRepo.CreateAuditLog(ctx.Context(), &models.AuditLog{
			ActorID:        actorID,
			ActorName:      actorEmail,
			ActorRole:      actorRole,
			Action:         "BLOCK_IP",
			TargetResource: "ip:" + cleanIP,
			Details:        reason,
			IPAddress:      callerIP,
			Category:       "SECURITY",
			Severity:       "WARN",
			StatusCode:     fiber.StatusCreated,
			UserAgent:      ctx.Get("User-Agent"),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Alamat IP %s berhasil diblokir", cleanIP),
		Data:    saved,
	})
}

func (c *SecurityController) UnblockIP(ctx *fiber.Ctx) error {
	target := strings.TrimSpace(ctx.Params("id"))
	if target == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "ID atau alamat IP wajib diisi",
		})
	}

	var err error
	if c.blockMgr != nil {
		err = c.blockMgr.UnblockIP(ctx.Context(), target)
	} else {
		err = c.secRepo.UnblockIP(ctx.Context(), target)
	}
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Aturan blokir IP aktif tidak ditemukan",
			})
		}
		return utils.ResponseInternalError(ctx, "Gagal membuka blokir alamat IP", err)
	}

	actorID, actorEmail, actorRole := resolveActorFromCtx(ctx)
	callerIP := ctx.IP()
	if c.blockMgr != nil {
		callerIP = c.blockMgr.ResolveClientIP(ctx)
	}
	if c.auditRepo != nil {
		_ = c.auditRepo.CreateAuditLog(ctx.Context(), &models.AuditLog{
			ActorID:        actorID,
			ActorName:      actorEmail,
			ActorRole:      actorRole,
			Action:         "UNBLOCK_IP",
			TargetResource: "ip:" + target,
			Details:        fmt.Sprintf("Membuka blokir IP/aturan %s", target),
			IPAddress:      callerIP,
			Category:       "SECURITY",
			Severity:       "INFO",
			StatusCode:     fiber.StatusOK,
			UserAgent:      ctx.Get("User-Agent"),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Blokir alamat IP berhasil dibuka",
	})
}
