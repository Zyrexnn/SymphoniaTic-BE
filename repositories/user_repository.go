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
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUserByID(ctx context.Context, id string) (*models.UserRecord, error) {
	var user models.UserRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, COALESCE(phone, ''), role, is_verified, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.Phone,
		&user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil profil pengguna: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.UserRecord, error) {
	var user models.UserRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, COALESCE(phone, ''), role, is_verified, created_at, updated_at
		FROM users WHERE LOWER(email) = LOWER($1)
	`, strings.TrimSpace(email)).Scan(
		&user.ID, &user.Email, &user.Name, &user.Phone,
		&user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal menemukan pengguna berdasarkan email: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetUserWithPassword(ctx context.Context, email string) (*models.UserRecord, string, error) {
	var user models.UserRecord
	var passwordHash string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, COALESCE(phone, ''), COALESCE(password_hash, ''), role, is_verified, created_at, updated_at
		FROM users WHERE LOWER(email) = LOWER($1)
	`, strings.TrimSpace(email)).Scan(
		&user.ID, &user.Email, &user.Name, &user.Phone, &passwordHash,
		&user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("gagal mengambil kredensial pengguna: %w", err)
	}

	return &user, passwordHash, nil
}

func (r *userRepository) GetPasswordHash(ctx context.Context, id string) (string, error) {
	var passwordHash string
	err := r.db.QueryRowContext(ctx, "SELECT COALESCE(password_hash, '') FROM users WHERE id = $1", id).Scan(&passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("gagal mengambil hash sandi: %w", err)
	}

	return passwordHash, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.UserRecord, passwordHash string) error {
	if user.ID == "" {
		user.ID = "usr-" + uuid.New().String()
	}
	if user.Role == "" {
		user.Role = "USER"
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, phone, password_hash, role, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, strings.ToLower(strings.TrimSpace(user.Email)), user.Name, user.Phone, passwordHash, user.Role, user.IsVerified, now, now)
	if err != nil {
		return fmt.Errorf("gagal membuat akun pengguna: %w", err)
	}

	return nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, id string, name, phone string) (*models.UserRecord, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE users SET name = $1, phone = $2, updated_at = NOW() WHERE id = $3
	`, strings.TrimSpace(name), strings.TrimSpace(phone), id)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui data profil: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa baris yang terpengaruh: %w", err)
	}
	if affected == 0 {
		return nil, ErrNotFound
	}

	return r.GetUserByID(ctx, id)
}

func (r *userRepository) UpdatePassword(ctx context.Context, id string, newPasswordHash string) error {
	result, err := r.db.ExecContext(ctx, "UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", newPasswordHash, id)
	if err != nil {
		return fmt.Errorf("gagal memperbarui kata sandi: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa baris perubahan kata sandi: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *userRepository) UpdatePasswordByEmail(ctx context.Context, email string, newPasswordHash string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $1, updated_at = NOW() WHERE LOWER(email) = LOWER($2)
	`, newPasswordHash, strings.TrimSpace(email))
	if err != nil {
		return fmt.Errorf("gagal mengatur ulang kata sandi: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa baris pengaturan ulang kata sandi: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *userRepository) CheckCooldown(ctx context.Context, email, purpose string, cooldown time.Duration) (bool, error) {
	var lastCreatedAt time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT created_at FROM auth_otps 
		WHERE LOWER(email) = LOWER($1) AND purpose = $2 AND created_at > $3 
		ORDER BY created_at DESC LIMIT 1
	`, strings.TrimSpace(email), purpose, time.Now().Add(-cooldown)).Scan(&lastCreatedAt)

	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("gagal memeriksa cooldown OTP: %w", err)
}

func (r *userRepository) IssueOTP(ctx context.Context, email, purpose, otpCode string, expiresAt time.Time, cooldown time.Duration) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	isCoolingDown, err := r.CheckCooldown(ctx, normalizedEmail, purpose, cooldown)
	if err != nil {
		return err
	}
	if isCoolingDown {
		return ErrOTPCooldown
	}

	// Menonaktifkan OTP aktif sebelumnya agar tidak terjadi benturan token verifikasi ganda
	_, _ = r.db.ExecContext(ctx, `
		UPDATE auth_otps SET is_used = TRUE 
		WHERE LOWER(email) = LOWER($1) AND purpose = $2 AND is_used = FALSE
	`, normalizedEmail, purpose)

	otpID := "otp-" + uuid.New().String()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO auth_otps (id, email, otp_code, purpose, attempts, is_used, expires_at, created_at)
		VALUES ($1, $2, $3, $4, 0, FALSE, $5, $6)
	`, otpID, normalizedEmail, otpCode, purpose, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("gagal menyimpan kode OTP ke database: %w", err)
	}

	return nil
}

func (r *userRepository) VerifyOTP(ctx context.Context, email, purpose, otpCode string, maxAttempts int) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	trimmedCode := strings.TrimSpace(otpCode)

	var record models.AuthOTPRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, otp_code, purpose, attempts, is_used, expires_at, created_at
		FROM auth_otps
		WHERE LOWER(email) = LOWER($1) AND purpose = $2 AND is_used = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, normalizedEmail, purpose).Scan(
		&record.ID, &record.Email, &record.OTPCode, &record.Purpose,
		&record.Attempts, &record.IsUsed, &record.ExpiresAt, &record.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrOTPInvalid
		}
		return fmt.Errorf("gagal memverifikasi kode OTP: %w", err)
	}

	// Membatasi batas maksimum percobaan salah untuk mencegah serangan brute force
	if record.Attempts >= maxAttempts {
		_, _ = r.db.ExecContext(ctx, "UPDATE auth_otps SET is_used = TRUE WHERE id = $1", record.ID)
		return ErrOTPMaxExceed
	}

	if record.OTPCode != trimmedCode {
		newAttempts := record.Attempts + 1
		isExhausted := newAttempts >= maxAttempts
		_, _ = r.db.ExecContext(ctx, "UPDATE auth_otps SET attempts = $1, is_used = $2 WHERE id = $3", newAttempts, isExhausted, record.ID)

		if isExhausted {
			return ErrOTPMaxExceed
		}
		return fmt.Errorf("sisa percobaan: %d", maxAttempts-newAttempts)
	}

	_, err = r.db.ExecContext(ctx, "UPDATE auth_otps SET is_used = TRUE WHERE id = $1", record.ID)
	if err != nil {
		return fmt.Errorf("gagal menandai OTP telah digunakan: %w", err)
	}

	return nil
}
