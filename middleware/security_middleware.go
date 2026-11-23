package middleware

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

const (
	defaultAutoBanThreshold = 15
	defaultAutoBanWindow    = 5 * time.Minute
	defaultAutoBanDuration  = 15 * time.Minute
)

var suspiciousProbeSegments = []string{
	"/.env",
	"/.git",
	"/wp-admin",
	"/wp-login",
	"/phpmyadmin",
	"/xmlrpc.php",
}

var suspiciousPayloadPatterns = []string{
	"union select",
	"sleep(",
	"benchmark(",
	"<script",
	"../",
	"..\\",
}

type IPBlocklistManager struct {
	secRepo   repositories.SecurityRepository
	auditRepo repositories.AdminUserRepository

	mu      sync.RWMutex
	blocked map[string]models.BlockedIP

	attemptsMu sync.Mutex
	attempts   map[string][]time.Time

	AutoBanThreshold int
	AutoBanWindow    time.Duration
	AutoBanDuration  time.Duration
}

func NewIPBlocklistManager(secRepo repositories.SecurityRepository, auditRepo repositories.AdminUserRepository) *IPBlocklistManager {
	return &IPBlocklistManager{
		secRepo:          secRepo,
		auditRepo:        auditRepo,
		blocked:          make(map[string]models.BlockedIP),
		attempts:         make(map[string][]time.Time),
		AutoBanThreshold: defaultAutoBanThreshold,
		AutoBanWindow:    defaultAutoBanWindow,
		AutoBanDuration:  defaultAutoBanDuration,
	}
}

func (m *IPBlocklistManager) RefreshFromDB(ctx context.Context) error {
	if m.secRepo == nil {
		return nil
	}
	list, err := m.secRepo.ListBlockedIPs(ctx, true)
	if err != nil {
		return err
	}

	next := make(map[string]models.BlockedIP, len(list))
	now := time.Now()
	for _, item := range list {
		if item.ExpiresAt != nil && now.After(*item.ExpiresAt) {
			continue
		}
		next[item.IPAddress] = item
	}

	m.mu.Lock()
	m.blocked = next
	m.mu.Unlock()
	return nil
}

func (m *IPBlocklistManager) BlockIP(ctx context.Context, entry *models.BlockedIP) (*models.BlockedIP, error) {
	if entry == nil {
		return nil, fmt.Errorf("data blokir IP tidak boleh kosong")
	}
	var saved *models.BlockedIP
	var err error
	if m.secRepo != nil {
		saved, err = m.secRepo.BlockIP(ctx, entry)
		if err != nil {
			return nil, err
		}
	} else {
		saved = entry
		saved.IsActive = true
		if saved.CreatedAt.IsZero() {
			saved.CreatedAt = time.Now()
		}
	}

	m.mu.Lock()
	m.blocked[saved.IPAddress] = *saved
	m.mu.Unlock()
	return saved, nil
}

func (m *IPBlocklistManager) UnblockIP(ctx context.Context, idOrIP string) error {
	if m.secRepo != nil {
		if err := m.secRepo.UnblockIP(ctx, idOrIP); err != nil {
			return err
		}
	}

	m.mu.Lock()
	delete(m.blocked, idOrIP)
	for ip, entry := range m.blocked {
		if entry.ID == idOrIP {
			delete(m.blocked, ip)
		}
	}
	m.mu.Unlock()
	return nil
}

// ResolveClientIP mengizinkan simulasi IP klien di lingkungan development melalui header X-Test-Client-IP
// agar pengujian blokir IP di localhost tidak mengunci sesi Super Admin.
func (m *IPBlocklistManager) ResolveClientIP(ctx *fiber.Ctx) string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "development") {
		if testIP := strings.TrimSpace(ctx.Get("X-Test-Client-IP")); testIP != "" {
			return testIP
		}
	}
	return ctx.IP()
}

func (m *IPBlocklistManager) isCachedBlocked(ip string) (models.BlockedIP, bool) {
	m.mu.RLock()
	entry, ok := m.blocked[ip]
	m.mu.RUnlock()
	if !ok {
		return models.BlockedIP{}, false
	}
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		m.mu.Lock()
		if current, stillExists := m.blocked[ip]; stillExists && current.ExpiresAt != nil && time.Now().After(*current.ExpiresAt) {
			delete(m.blocked, ip)
		}
		m.mu.Unlock()
		return models.BlockedIP{}, false
	}
	return entry, true
}

func (m *IPBlocklistManager) BlocklistMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		clientIP := m.ResolveClientIP(ctx)
		if entry, blocked := m.isCachedBlocked(clientIP); blocked {
			return ctx.Status(fiber.StatusForbidden).JSON(models.APIResponse{
				Success: false,
				Message: fmt.Sprintf("Akses ditolak. Alamat IP Anda sedang diblokir oleh sistem keamanan SymphoniaTic (%s).", entry.Reason),
			})
		}
		return ctx.Next()
	}
}

func isSuspiciousProbe(ctx *fiber.Ctx) bool {
	lowerPath := strings.ToLower(ctx.Path())
	for _, seg := range suspiciousProbeSegments {
		if strings.Contains(lowerPath, seg) {
			return true
		}
	}
	rawQuery := strings.ToLower(string(ctx.Request().URI().QueryString()))
	for _, pat := range suspiciousPayloadPatterns {
		if strings.Contains(lowerPath, pat) || strings.Contains(rawQuery, pat) {
			return true
		}
	}
	return false
}

func (m *IPBlocklistManager) recordFailedAttempt(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-m.AutoBanWindow)

	m.attemptsMu.Lock()
	defer m.attemptsMu.Unlock()

	history := m.attempts[ip]
	pruned := history[:0]
	for _, ts := range history {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}
	pruned = append(pruned, now)
	m.attempts[ip] = pruned

	if len(pruned) >= m.AutoBanThreshold {
		delete(m.attempts, ip)
		return true
	}
	return false
}

func (m *IPBlocklistManager) SecurityAuditMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		err := ctx.Next()

		status := ctx.Response().StatusCode()
		probe := isSuspiciousProbe(ctx)
		if status != fiber.StatusUnauthorized &&
			status != fiber.StatusForbidden &&
			status != fiber.StatusTooManyRequests &&
			!probe {
			return err
		}

		clientIP := m.ResolveClientIP(ctx)
		path := ctx.Path()
		userAgent := ctx.Get("User-Agent")

		category := "SECURITY"
		if strings.Contains(path, "/auth/") || strings.HasSuffix(path, "/admin/login") {
			category = "AUTH"
		}

		severity := "WARN"
		action := "SUSPICIOUS_REQUEST"
		switch {
		case probe:
			severity = "CRITICAL"
			action = "SECURITY_PROBE_DETECTED"
		case status == fiber.StatusTooManyRequests:
			severity = "CRITICAL"
			action = "RATE_LIMIT_EXCEEDED"
		case status == fiber.StatusUnauthorized:
			action = "UNAUTHORIZED_ACCESS"
		case status == fiber.StatusForbidden:
			action = "FORBIDDEN_ACCESS"
		}

		actorID := "anonymous"
		actorName := clientIP
		actorRole := "ANONYMOUS"
		if id, ok := ctx.Locals("admin_id").(string); ok && id != "" {
			actorID = id
		} else if id, ok := ctx.Locals("user_id").(string); ok && id != "" {
			actorID = id
			actorRole = "USER"
		}
		if email, ok := ctx.Locals("admin_email").(string); ok && email != "" {
			actorName = email
		} else if email, ok := ctx.Locals("user_email").(string); ok && email != "" {
			actorName = email
		}
		if role, ok := ctx.Locals("admin_role").(string); ok && role != "" {
			actorRole = role
		}

		if m.auditRepo != nil {
			_ = m.auditRepo.CreateAuditLog(ctx.Context(), &models.AuditLog{
				ActorID:        actorID,
				ActorName:      actorName,
				ActorRole:      actorRole,
				Action:         action,
				TargetResource: ctx.Method() + " " + path,
				Details:        fmt.Sprintf("%s %s merespons HTTP %d", ctx.Method(), path, status),
				IPAddress:      clientIP,
				Category:       category,
				Severity:       severity,
				StatusCode:     status,
				UserAgent:      userAgent,
			})
		}

		if (status == fiber.StatusUnauthorized || (status == fiber.StatusForbidden && actorRole == "ANONYMOUS") || probe) && m.recordFailedAttempt(clientIP) {
			expires := time.Now().Add(m.AutoBanDuration)
			reason := "Auto-ban: Terdeteksi percobaan autentikasi/akses gagal berulang (Brute Force)"
			if probe {
				reason = "Auto-ban: Terdeteksi pemindaian kerentanan (Security Probe)"
			}
			if _, banErr := m.BlockIP(ctx.Context(), &models.BlockedIP{
				IPAddress: clientIP,
				Reason:    reason,
				BlockedBy: "SYSTEM_AUTO_DEFENSE",
				IsAuto:    true,
				ExpiresAt: &expires,
			}); banErr == nil && m.auditRepo != nil {
				_ = m.auditRepo.CreateAuditLog(ctx.Context(), &models.AuditLog{
					ActorID:        "system",
					ActorName:      "SYSTEM_AUTO_DEFENSE",
					ActorRole:      "SYSTEM",
					Action:         "AUTO_BLOCK_IP",
					TargetResource: "ip:" + clientIP,
					Details:        reason,
					IPAddress:      clientIP,
					Category:       "SECURITY",
					Severity:       "CRITICAL",
					StatusCode:     status,
					UserAgent:      userAgent,
				})
			}
		}

		return err
	}
}
