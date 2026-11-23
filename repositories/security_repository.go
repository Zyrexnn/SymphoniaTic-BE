package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/google/uuid"
)

type securityRepository struct {
	db *sql.DB
}

func NewSecurityRepository(db *sql.DB) SecurityRepository {
	return &securityRepository{db: db}
}

func buildAuditFilterWhere(filter models.AuditLogFilter) (string, []interface{}) {
	where := "WHERE 1=1"
	args := make([]interface{}, 0, 8)
	idx := 1

	if v := strings.ToUpper(strings.TrimSpace(filter.Action)); v != "" && v != "ALL" {
		where += fmt.Sprintf(" AND UPPER(action) = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v := strings.ToUpper(strings.TrimSpace(filter.Category)); v != "" && v != "ALL" {
		where += fmt.Sprintf(" AND UPPER(category) = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v := strings.ToUpper(strings.TrimSpace(filter.Severity)); v != "" && v != "ALL" {
		where += fmt.Sprintf(" AND UPPER(severity) = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v := strings.ToUpper(strings.TrimSpace(filter.ActorRole)); v != "" && v != "ALL" {
		where += fmt.Sprintf(" AND UPPER(actor_role) = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v := strings.TrimSpace(filter.IPAddress); v != "" {
		where += fmt.Sprintf(" AND ip_address LIKE $%d", idx)
		args = append(args, "%"+v+"%")
		idx++
	}
	if filter.From != nil && !filter.From.IsZero() {
		where += fmt.Sprintf(" AND created_at >= $%d", idx)
		args = append(args, *filter.From)
		idx++
	}
	if filter.To != nil && !filter.To.IsZero() {
		where += fmt.Sprintf(" AND created_at <= $%d", idx)
		args = append(args, *filter.To)
		idx++
	}
	if v := strings.TrimSpace(filter.Search); v != "" {
		where += fmt.Sprintf(
			" AND (LOWER(actor_name) LIKE LOWER($%d) OR LOWER(target_resource) LIKE LOWER($%d) OR LOWER(details) LIKE LOWER($%d) OR LOWER(action) LIKE LOWER($%d) OR LOWER(ip_address) LIKE LOWER($%d))",
			idx, idx, idx, idx, idx,
		)
		args = append(args, "%"+v+"%")
	}

	return where, args
}

func (r *securityRepository) GetAuditLogsFiltered(ctx context.Context, filter models.AuditLogFilter) (*models.AuditLogPage, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	whereClause, args := buildAuditFilterWhere(filter)

	countQuery := "SELECT COUNT(*) FROM audit_logs " + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("gagal menghitung total log audit: %w", err)
	}

	offset := (page - 1) * limit
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2

	dataQuery := fmt.Sprintf(`
		SELECT id, actor_id, actor_name, actor_role, action, target_resource,
		       COALESCE(details, ''), COALESCE(ip_address, ''),
		       COALESCE(category, 'OPERATIONAL'), COALESCE(severity, 'INFO'),
		       COALESCE(status_code, 200), COALESCE(user_agent, ''), created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, limitIdx, offsetIdx)

	queryArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat log audit: %w", err)
	}
	defer rows.Close()

	items := make([]models.AuditLog, 0)
	for rows.Next() {
		var entry models.AuditLog
		if err := rows.Scan(
			&entry.ID, &entry.ActorID, &entry.ActorName, &entry.ActorRole,
			&entry.Action, &entry.TargetResource, &entry.Details, &entry.IPAddress,
			&entry.Category, &entry.Severity, &entry.StatusCode, &entry.UserAgent, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris log audit: %w", err)
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi log audit: %w", err)
	}

	return &models.AuditLogPage{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *securityRepository) IsIPBlocked(ctx context.Context, ip string) (*models.BlockedIP, error) {
	cleanIP := strings.TrimSpace(ip)
	if cleanIP == "" {
		return nil, ErrNotFound
	}

	var entry models.BlockedIP
	var expiresAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, ip_address, reason, blocked_by, is_auto, is_active, expires_at, created_at
		FROM blocked_ips
		WHERE ip_address = $1
		  AND is_active = TRUE
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY created_at DESC
		LIMIT 1
	`, cleanIP).Scan(
		&entry.ID, &entry.IPAddress, &entry.Reason, &entry.BlockedBy,
		&entry.IsAuto, &entry.IsActive, &expiresAt, &entry.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa status blokir IP: %w", err)
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		entry.ExpiresAt = &t
	}
	return &entry, nil
}

func (r *securityRepository) BlockIP(ctx context.Context, entry *models.BlockedIP) (*models.BlockedIP, error) {
	if entry == nil || strings.TrimSpace(entry.IPAddress) == "" {
		return nil, fmt.Errorf("alamat IP wajib diisi")
	}

	cleanIP := strings.TrimSpace(entry.IPAddress)
	id := entry.ID
	if id == "" {
		id = "blk-" + uuid.New().String()
	}
	now := entry.CreatedAt
	if now.IsZero() {
		now = time.Now()
	}

	// Nonaktifkan blokir lama untuk IP yang sama agar hanya ada satu aturan aktif per IP
	if _, err := r.db.ExecContext(ctx, `UPDATE blocked_ips SET is_active = FALSE WHERE ip_address = $1 AND is_active = TRUE`, cleanIP); err != nil {
		return nil, fmt.Errorf("gagal memperbarui blokir IP lama: %w", err)
	}

	var expiresParam interface{}
	if entry.ExpiresAt != nil {
		expiresParam = *entry.ExpiresAt
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO blocked_ips (id, ip_address, reason, blocked_by, is_auto, is_active, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6, $7)
	`, id, cleanIP, entry.Reason, entry.BlockedBy, entry.IsAuto, expiresParam, now)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan blokir IP: %w", err)
	}

	return &models.BlockedIP{
		ID:        id,
		IPAddress: cleanIP,
		Reason:    entry.Reason,
		BlockedBy: entry.BlockedBy,
		IsAuto:    entry.IsAuto,
		IsActive:  true,
		ExpiresAt: entry.ExpiresAt,
		CreatedAt: now,
	}, nil
}

func (r *securityRepository) UnblockIP(ctx context.Context, idOrIP string) error {
	clean := strings.TrimSpace(idOrIP)
	if clean == "" {
		return ErrNotFound
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE blocked_ips
		SET is_active = FALSE
		WHERE (id = $1 OR ip_address = $1) AND is_active = TRUE
	`, clean)
	if err != nil {
		return fmt.Errorf("gagal membuka blokir IP: %w", err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *securityRepository) ListBlockedIPs(ctx context.Context, activeOnly bool) ([]models.BlockedIP, error) {
	query := `
		SELECT id, ip_address, reason, blocked_by, is_auto, is_active, expires_at, created_at
		FROM blocked_ips
	`
	if activeOnly {
		query += ` WHERE is_active = TRUE AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`
	}
	query += ` ORDER BY created_at DESC LIMIT 200`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar IP diblokir: %w", err)
	}
	defer rows.Close()

	list := make([]models.BlockedIP, 0)
	for rows.Next() {
		var entry models.BlockedIP
		var expiresAt sql.NullTime
		if err := rows.Scan(
			&entry.ID, &entry.IPAddress, &entry.Reason, &entry.BlockedBy,
			&entry.IsAuto, &entry.IsActive, &expiresAt, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris IP diblokir: %w", err)
		}
		if expiresAt.Valid {
			t := expiresAt.Time
			entry.ExpiresAt = &t
		}
		list = append(list, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi daftar IP diblokir: %w", err)
	}
	return list, nil
}
