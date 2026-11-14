package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/google/uuid"
)

type refundRepository struct {
	db *sql.DB
}

func NewRefundRepository(db *sql.DB) RefundRepository {
	return &refundRepository{db: db}
}

func (r *refundRepository) RequestRefundOTP(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error {
	var existingRefundID string
	var inCooldown bool

	// Memeriksa keberadaan permohonan refund sebelumnya serta mengecek cooldown 60 detik untuk mencegah spam OTP
	err := r.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(updated_at > NOW() - INTERVAL '60 seconds', false) AS in_cooldown
		FROM refund_requests
		WHERE order_id = $1
	`, orderID).Scan(&existingRefundID, &inCooldown)

	if errors.Is(err, sql.ErrNoRows) {
		refundID := "rf-" + uuid.New().String()
		_, err = r.db.ExecContext(ctx, `
			INSERT INTO refund_requests (id, order_id, order_code, user_email, bank_name, account_number, account_holder, reason, refund_amount, status, otp_code, otp_expires_at, attempts, created_at, updated_at)
			VALUES ($1, $2, $3, $4, '', '', '', '', $5, 'OTP_SENT', $6, $7, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, refundID, orderID, orderCode, userEmail, refundAmount, otpCode, expiresAt)
		if err != nil {
			return fmt.Errorf("gagal membuat permintaan refund baru: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("gagal memeriksa data refund: %w", err)
	}

	if inCooldown {
		return ErrOTPCooldown
	}

	// Me-reset hitungan percobaan ke 0 dan menerbitkan OTP baru beserta waktu kedaluwarsa baru
	_, err = r.db.ExecContext(ctx, `
		UPDATE refund_requests
		SET otp_code = $1, otp_expires_at = $2, attempts = 0, status = 'OTP_SENT', updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, otpCode, expiresAt, existingRefundID)
	if err != nil {
		return fmt.Errorf("gagal memperbarui kode OTP refund: %w", err)
	}

	return nil
}

func (r *refundRepository) GetRefundByOrderCode(ctx context.Context, orderCode string) (*models.RefundRequestRecord, error) {
	var record models.RefundRequestRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, r.account_holder,
		       COALESCE(r.reason, ''), r.refund_amount, r.status, COALESCE(r.admin_note, ''), r.created_at, r.updated_at,
		       COALESCE(o.event_title, ''), COALESCE(o.category_name, ''), COALESCE(o.quantity, 0), COALESCE(o.user_name, '')
		FROM refund_requests r
		LEFT JOIN orders o ON o.id = r.order_id
		WHERE UPPER(r.order_code) = UPPER($1)
	`, orderCode).Scan(
		&record.ID, &record.OrderID, &record.OrderCode, &record.UserEmail, &record.BankName,
		&record.AccountNumber, &record.AccountHolder, &record.Reason, &record.RefundAmount,
		&record.Status, &record.AdminNote, &record.CreatedAt, &record.UpdatedAt,
		&record.EventTitle, &record.CategoryName, &record.Quantity, &record.UserName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data refund: %w", err)
	}

	return &record, nil
}

func (r *refundRepository) GetRefundWithOTP(ctx context.Context, orderCode string) (*models.RefundRequestRecord, string, time.Time, error) {
	var record models.RefundRequestRecord
	var otpCode string
	var otpExpiresAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT id, order_id, order_code, user_email, status, COALESCE(otp_code, ''), COALESCE(otp_expires_at, '1970-01-01'::timestamptz)
		FROM refund_requests
		WHERE UPPER(order_code) = UPPER($1)
	`, orderCode).Scan(&record.ID, &record.OrderID, &record.OrderCode, &record.UserEmail, &record.Status, &otpCode, &otpExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", time.Time{}, ErrNotFound
		}
		return nil, "", time.Time{}, fmt.Errorf("gagal mengambil data refund dan OTP: %w", err)
	}

	return &record, otpCode, otpExpiresAt, nil
}

func (r *refundRepository) GetRefundStatus(ctx context.Context, orderCode, userEmail string) (*models.RefundRequestRecord, string, error) {
	var record models.RefundRequestRecord
	var orderStatus string

	err := r.db.QueryRowContext(ctx, `
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, r.account_holder, 
		       r.reason, r.refund_amount, r.status, COALESCE(r.admin_note, ''), r.created_at, r.updated_at, 
		       o.status, o.event_title, o.category_name, o.quantity, o.user_name
		FROM refund_requests r
		JOIN orders o ON o.id = r.order_id
		WHERE UPPER(r.order_code) = UPPER($1) AND LOWER(r.user_email) = LOWER($2)
	`, orderCode, userEmail).Scan(
		&record.ID, &record.OrderID, &record.OrderCode, &record.UserEmail, &record.BankName,
		&record.AccountNumber, &record.AccountHolder, &record.Reason, &record.RefundAmount,
		&record.Status, &record.AdminNote, &record.CreatedAt, &record.UpdatedAt,
		&orderStatus, &record.EventTitle, &record.CategoryName, &record.Quantity, &record.UserName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("gagal mengambil status refund: %w", err)
	}

	return &record, orderStatus, nil
}

func (r *refundRepository) SubmitRefund(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi pengajuan refund: %w", err)
	}
	defer tx.Rollback()

	var record models.RefundRequestRecord
	var storedOTP string
	var expiresAt time.Time
	var attempts int

	// Mengunci baris refund_requests dengan FOR UPDATE untuk memastikan konsistensi counter percobaan (attempts) dan validasi OTP
	err = tx.QueryRowContext(ctx, `
		SELECT id, order_id, order_code, user_email, status, 
		       COALESCE(otp_code, ''), COALESCE(otp_expires_at, '1970-01-01'::timestamptz), COALESCE(attempts, 0)
		FROM refund_requests
		WHERE UPPER(order_code) = UPPER($1)
		FOR UPDATE
	`, input.OrderCode).Scan(&record.ID, &record.OrderID, &record.OrderCode, &record.UserEmail, &record.Status, &storedOTP, &expiresAt, &attempts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data refund dan OTP: %w", err)
	}

	if !strings.EqualFold(record.UserEmail, input.UserEmail) {
		return nil, ErrUnauthorized
	}

	// Membatasi toleransi kesalahan OTP maksimal 5 kali untuk mencegah serangan brute force
	if attempts >= 5 {
		_, _ = tx.ExecContext(ctx, `
			UPDATE refund_requests
			SET otp_code = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, record.ID)
		_ = tx.Commit()
		return nil, ErrOTPMaxExceed
	}

	if storedOTP == "" || time.Now().After(expiresAt) {
		return nil, ErrOTPInvalid
	}

	if storedOTP != input.OTPCode {
		newAttempts := attempts + 1
		remaining := 5 - newAttempts
		if remaining <= 0 {
			_, _ = tx.ExecContext(ctx, `
				UPDATE refund_requests
				SET attempts = $1, otp_code = NULL, updated_at = CURRENT_TIMESTAMP
				WHERE id = $2
			`, newAttempts, record.ID)
			_ = tx.Commit()
			return nil, ErrOTPMaxExceed
		}

		_, _ = tx.ExecContext(ctx, `
			UPDATE refund_requests
			SET attempts = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, newAttempts, record.ID)
		_ = tx.Commit()
		return nil, &ErrOTPWrongCode{RemainingAttempts: remaining}
	}

	// Menghapus nilai otp_code dan mereset attempts ke 0 setelah verifikasi berhasil guna mencegah penggunaan berulang
	_, err = tx.ExecContext(ctx, `
		UPDATE refund_requests
		SET bank_name = $1, account_number = $2, account_holder = $3, reason = $4, status = 'PENDING', otp_code = NULL, attempts = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`, input.BankName, input.AccountNumber, input.AccountHolder, input.Reason, record.ID)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui permohonan refund: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE orders
		SET status = 'REFUND_REQUESTED'
		WHERE id = $1
	`, record.OrderID)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui status pesanan menjadi refund: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal konfirmasi transaksi pengajuan refund: %w", err)
	}

	record.BankName = input.BankName
	record.AccountNumber = input.AccountNumber
	record.AccountHolder = input.AccountHolder
	record.Reason = input.Reason
	record.Status = "PENDING"
	record.UpdatedAt = time.Now()

	return &record, nil
}

func (r *refundRepository) GetAllRefunds(ctx context.Context) ([]models.RefundRequestRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, r.account_holder, 
		       COALESCE(r.reason, ''), r.refund_amount, r.status, COALESCE(r.admin_note, ''), r.created_at, r.updated_at, 
		       COALESCE(o.event_title, ''), COALESCE(o.category_name, ''), COALESCE(o.quantity, 0), COALESCE(o.user_name, '')
		FROM refund_requests r
		JOIN orders o ON o.id = r.order_id
		WHERE r.status != 'OTP_SENT'
		ORDER BY r.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil seluruh daftar refund: %w", err)
	}
	defer rows.Close()

	refundList := make([]models.RefundRequestRecord, 0)
	for rows.Next() {
		var record models.RefundRequestRecord
		if err := rows.Scan(
			&record.ID, &record.OrderID, &record.OrderCode, &record.UserEmail, &record.BankName,
			&record.AccountNumber, &record.AccountHolder, &record.Reason, &record.RefundAmount,
			&record.Status, &record.AdminNote, &record.CreatedAt, &record.UpdatedAt,
			&record.EventTitle, &record.CategoryName, &record.Quantity, &record.UserName,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris data refund: %w", err)
		}
		refundList = append(refundList, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris refund: %w", err)
	}

	return refundList, nil
}

func (r *refundRepository) GetUserRefunds(ctx context.Context, userEmail string) ([]models.RefundRequestRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, 
		       r.account_holder, COALESCE(r.reason, ''), r.refund_amount, r.status, 
		       COALESCE(r.admin_note, ''), r.created_at, r.updated_at,
		       COALESCE(o.event_title, ''), COALESCE(o.category_name, ''), 
		       COALESCE(o.quantity, 0), COALESCE(o.user_name, '')
		FROM refund_requests r
		LEFT JOIN orders o ON r.order_id = o.id
		WHERE LOWER(r.user_email) = LOWER($1)
		ORDER BY r.created_at DESC
	`, userEmail)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat refund pengguna: %w", err)
	}
	defer rows.Close()

	userRefunds := make([]models.RefundRequestRecord, 0)
	for rows.Next() {
		var record models.RefundRequestRecord
		if err := rows.Scan(
			&record.ID, &record.OrderID, &record.OrderCode, &record.UserEmail, &record.BankName,
			&record.AccountNumber, &record.AccountHolder, &record.Reason, &record.RefundAmount,
			&record.Status, &record.AdminNote, &record.CreatedAt, &record.UpdatedAt,
			&record.EventTitle, &record.CategoryName, &record.Quantity, &record.UserName,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris data refund pengguna: %w", err)
		}
		userRefunds = append(userRefunds, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris refund pengguna: %w", err)
	}

	return userRefunds, nil
}

func (r *refundRepository) UpdateRefundStatus(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error) {
	normalizedStatus := strings.ToUpper(strings.TrimSpace(newStatus))
	if normalizedStatus != "APPROVED" && normalizedStatus != "REJECTED" && normalizedStatus != "COMPLETED" {
		return nil, fmt.Errorf("status refund tidak valid: %s", newStatus)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi pembaruan status refund: %w", err)
	}
	defer tx.Rollback()

	// Mengunci baris refund_requests dan orders menggunakan FOR UPDATE untuk mencegah modifikasi konkruen status refund dan penggandaan kuota
	var orderID, orderCode, userEmail, currentStatus, categoryName, eventID string
	var quantity int
	var amount float64

	err = tx.QueryRowContext(ctx, `
		SELECT r.order_id, r.order_code, r.user_email, r.refund_amount, r.status, o.event_id, o.category_name, o.quantity
		FROM refund_requests r
		JOIN orders o ON o.id = r.order_id
		WHERE r.id = $1
		FOR UPDATE OF r, o
	`, refundID).Scan(&orderID, &orderCode, &userEmail, &amount, &currentStatus, &eventID, &categoryName, &quantity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil rincian refund: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE refund_requests
		SET status = $1, admin_note = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, normalizedStatus, adminNote, refundID)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui status refund di database: %w", err)
	}

	if normalizedStatus == "APPROVED" || normalizedStatus == "COMPLETED" {
		_, err = tx.ExecContext(ctx, "UPDATE orders SET status = 'REFUNDED' WHERE id = $1", orderID)
		if err != nil {
			return nil, fmt.Errorf("gagal memperbarui status pesanan menjadi REFUNDED: %w", err)
		}

		// Pengembalian kuota tiket hanya dilakukan jika status refund bertransisi dari PENDING guna mencegah bug penggandaan kuota saat status diubah dari APPROVED ke COMPLETED
		if currentStatus == "PENDING" {
			res, err := tx.ExecContext(ctx, `
				UPDATE ticket_categories
				SET remaining_quota = remaining_quota + $1
				WHERE event_id = $2 AND name = $3
			`, quantity, eventID, categoryName)
			if err != nil {
				return nil, fmt.Errorf("gagal mengembalikan kuota tiket: %w", err)
			}
			rowsAffected, err := res.RowsAffected()
			if err == nil && rowsAffected == 0 {
				log.Printf("Peringatan: Kategori tiket %s pada event %s tidak ditemukan saat pengembalian kuota refund %s", categoryName, eventID, refundID)
			}
		}
	} else if normalizedStatus == "REJECTED" {
		// Mengembalikan status pesanan ke ISSUED jika permohonan ditolak agar tiket tetap valid dan muncul di dashboard
		_, err = tx.ExecContext(ctx, "UPDATE orders SET status = 'ISSUED' WHERE id = $1", orderID)
		if err != nil {
			return nil, fmt.Errorf("gagal mengembalikan status pesanan ke ISSUED: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal konfirmasi transaksi perubahan status refund: %w", err)
	}

	return &models.RefundRequestRecord{
		ID:           refundID,
		OrderID:      orderID,
		OrderCode:    orderCode,
		UserEmail:    userEmail,
		RefundAmount: amount,
		Status:       normalizedStatus,
		AdminNote:    adminNote,
		CategoryName: categoryName,
		Quantity:     quantity,
		UpdatedAt:    time.Now(),
	}, nil
}
