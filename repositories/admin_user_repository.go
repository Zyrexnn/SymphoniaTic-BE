package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type adminUserRepository struct {
	db *sql.DB
}

func NewAdminUserRepository(db *sql.DB) AdminUserRepository {
	return &adminUserRepository{db: db}
}

const adminUserSelectColumns = `
	SELECT id, username, name, email, password_hash, role, is_active, created_at, updated_at
	FROM admin_users
`

func scanAdminUser(scanner rowScanner) (models.AdminUser, error) {
	var u models.AdminUser
	err := scanner.Scan(
		&u.ID, &u.Username, &u.Name, &u.Email, &u.PasswordHash,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

func (r *adminUserRepository) GetByUsername(ctx context.Context, username string) (*models.AdminUser, error) {
	clean := strings.TrimSpace(username)
	if clean == "" {
		return nil, ErrNotFound
	}
	u, err := scanAdminUser(r.db.QueryRowContext(ctx, adminUserSelectColumns+`
		WHERE LOWER(username) = LOWER($1)
	`, clean))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data staf berdasarkan username: %w", err)
	}
	return &u, nil
}

func (r *adminUserRepository) GetByID(ctx context.Context, id string) (*models.AdminUser, error) {
	u, err := scanAdminUser(r.db.QueryRowContext(ctx, adminUserSelectColumns+`
		WHERE id = $1
	`, strings.TrimSpace(id)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data staf berdasarkan ID: %w", err)
	}
	return &u, nil
}

func (r *adminUserRepository) GetAllStaff(ctx context.Context, roleFilter string, search string) ([]models.AdminUser, error) {
	query := adminUserSelectColumns + " WHERE 1=1"
	args := make([]interface{}, 0, 2)
	idx := 1

	cleanRole := strings.ToUpper(strings.TrimSpace(roleFilter))
	if cleanRole != "" && cleanRole != "ALL" {
		query += fmt.Sprintf(" AND role = $%d", idx)
		args = append(args, cleanRole)
		idx++
	}

	cleanSearch := strings.TrimSpace(search)
	if cleanSearch != "" {
		query += fmt.Sprintf(" AND (LOWER(username) LIKE LOWER($%d) OR LOWER(name) LIKE LOWER($%d) OR LOWER(email) LIKE LOWER($%d))", idx, idx, idx)
		args = append(args, "%"+cleanSearch+"%")
		idx++
	}

	query += " ORDER BY CASE role WHEN 'SUPER_ADMIN' THEN 1 WHEN 'ADMIN' THEN 2 ELSE 3 END, created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar akun staf: %w", err)
	}
	defer rows.Close()

	staffList := make([]models.AdminUser, 0)
	for rows.Next() {
		u, err := scanAdminUser(rows)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca baris akun staf: %w", err)
		}
		staffList = append(staffList, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi daftar akun staf: %w", err)
	}
	return staffList, nil
}

func (r *adminUserRepository) CreateStaff(ctx context.Context, req *models.CreateStaffRequest, passwordHash string) (*models.AdminUser, error) {
	id := "staff-" + uuid.New().String()
	now := time.Now()
	username := strings.ToLower(strings.TrimSpace(req.Username))
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)
	role := strings.ToUpper(strings.TrimSpace(req.Role))
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO admin_users (id, username, name, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, username, name, email, passwordHash, role, isActive, now, now)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrStaffDuplicateUsername
		}
		return nil, fmt.Errorf("gagal menyimpan akun staf baru: %w", err)
	}

	return &models.AdminUser{
		ID:        id,
		Username:  username,
		Name:      name,
		Email:     email,
		Role:      role,
		IsActive:  isActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *adminUserRepository) UpdateStaff(ctx context.Context, id string, req *models.UpdateStaffRequest) (*models.AdminUser, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	role := strings.ToUpper(strings.TrimSpace(req.Role))

	u, err := scanAdminUser(r.db.QueryRowContext(ctx, `
		UPDATE admin_users
		SET name = $1, email = $2, role = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, username, name, email, password_hash, role, is_active, created_at, updated_at
	`, name, email, role, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrStaffDuplicateUsername
		}
		return nil, fmt.Errorf("gagal memperbarui akun staf: %w", err)
	}
	return &u, nil
}

func (r *adminUserRepository) ToggleStaffActive(ctx context.Context, id string) (*models.AdminUser, error) {
	u, err := scanAdminUser(r.db.QueryRowContext(ctx, `
		UPDATE admin_users
		SET is_active = NOT is_active, updated_at = NOW()
		WHERE id = $1
		RETURNING id, username, name, email, password_hash, role, is_active, created_at, updated_at
	`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengubah status aktif akun staf: %w", err)
	}
	return &u, nil
}

func (r *adminUserRepository) ResetStaffPassword(ctx context.Context, id string, passwordHash string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE admin_users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("gagal mengatur ulang kata sandi staf: %w", err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *adminUserRepository) DeleteStaff(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM admin_users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("gagal menghapus akun staf: %w", err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *adminUserRepository) CountActiveSuperAdmins(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM admin_users WHERE role = 'SUPER_ADMIN' AND is_active = TRUE
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung jumlah Super Admin aktif: %w", err)
	}
	return count, nil
}

func (r *adminUserRepository) CreateAuditLog(ctx context.Context, logEntry *models.AuditLog) error {
	if logEntry == nil {
		return nil
	}
	id := logEntry.ID
	if id == "" {
		id = "audit-" + uuid.New().String()
	}
	now := logEntry.CreatedAt
	if now.IsZero() {
		now = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, actor_id, actor_name, actor_role, action, target_resource, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, logEntry.ActorID, logEntry.ActorName, logEntry.ActorRole, logEntry.Action, logEntry.TargetResource, logEntry.Details, logEntry.IPAddress, now)
	if err != nil {
		return fmt.Errorf("gagal mencatat audit log: %w", err)
	}
	return nil
}

func (r *adminUserRepository) GetAuditLogs(ctx context.Context, actionFilter, search string, limit int) ([]models.AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	query := `
		SELECT id, actor_id, actor_name, actor_role, action, target_resource, COALESCE(details, ''), COALESCE(ip_address, ''), created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := make([]interface{}, 0, 3)
	idx := 1

	cleanAction := strings.ToUpper(strings.TrimSpace(actionFilter))
	if cleanAction != "" && cleanAction != "ALL" {
		query += fmt.Sprintf(" AND UPPER(action) = $%d", idx)
		args = append(args, cleanAction)
		idx++
	}

	cleanSearch := strings.TrimSpace(search)
	if cleanSearch != "" {
		query += fmt.Sprintf(" AND (LOWER(actor_name) LIKE LOWER($%d) OR LOWER(target_resource) LIKE LOWER($%d) OR LOWER(details) LIKE LOWER($%d) OR LOWER(action) LIKE LOWER($%d))", idx, idx, idx, idx)
		args = append(args, "%"+cleanSearch+"%")
		idx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", idx)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat audit log: %w", err)
	}
	defer rows.Close()

	logs := make([]models.AuditLog, 0)
	for rows.Next() {
		var entry models.AuditLog
		if err := rows.Scan(
			&entry.ID, &entry.ActorID, &entry.ActorName, &entry.ActorRole,
			&entry.Action, &entry.TargetResource, &entry.Details, &entry.IPAddress, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris audit log: %w", err)
		}
		logs = append(logs, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi audit log: %w", err)
	}
	return logs, nil
}
