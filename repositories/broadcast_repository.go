package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/google/uuid"
)

type broadcastRepository struct {
	db *sql.DB
}

func NewBroadcastRepository(db *sql.DB) BroadcastRepository {
	return &broadcastRepository{db: db}
}

func (r *broadcastRepository) CreateBroadcast(ctx context.Context, record *models.EmailBroadcastRecord) error {
	var targetEventID interface{}
	if strings.TrimSpace(record.TargetEventID) != "" {
		targetEventID = strings.TrimSpace(record.TargetEventID)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO email_broadcasts (
			id, target_scope, target_event_id, target_category_id, category, subject, headline, body_text,
			cta_label, cta_url, promo_code, total_recipients, sent_count, failed_count,
			status, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`,
		record.ID,
		record.TargetScope,
		targetEventID,
		strings.TrimSpace(record.TargetCategoryID),
		record.Category,
		record.Subject,
		record.Headline,
		record.BodyText,
		record.CTALabel,
		record.CTAURL,
		record.PromoCode,
		record.TotalRecipients,
		record.SentCount,
		record.FailedCount,
		record.Status,
		record.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("gagal menyimpan catatan broadcast email: %w", err)
	}
	return nil
}

func (r *broadcastRepository) UpdateBroadcastProgress(ctx context.Context, id string, sentCount, failedCount int, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE email_broadcasts
		SET sent_count = $2,
		    failed_count = $3,
		    status = $4,
		    completed_at = CASE WHEN $4 IN ('COMPLETED', 'PARTIAL_FAILED', 'FAILED') THEN NOW() ELSE completed_at END
		WHERE id = $1
	`, id, sentCount, failedCount, status)
	if err != nil {
		return fmt.Errorf("gagal memperbarui progres broadcast email: %w", err)
	}
	return nil
}

func (r *broadcastRepository) GetAllBroadcasts(ctx context.Context) ([]models.EmailBroadcastRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.target_scope, COALESCE(b.target_event_id, ''), COALESCE(e.title, ''),
		       COALESCE(b.target_category_id, ''), b.category, b.subject, b.headline, b.body_text,
		       COALESCE(b.cta_label, ''), COALESCE(b.cta_url, ''), COALESCE(b.promo_code, ''),
		       b.total_recipients, b.sent_count, b.failed_count, b.status, b.created_by,
		       b.created_at, b.completed_at
		FROM email_broadcasts b
		LEFT JOIN events e ON b.target_event_id = e.id
		ORDER BY b.created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat broadcast email: %w", err)
	}
	defer rows.Close()

	list := make([]models.EmailBroadcastRecord, 0)
	for rows.Next() {
		var rec models.EmailBroadcastRecord
		var completedAt sql.NullTime
		if err := rows.Scan(
			&rec.ID,
			&rec.TargetScope,
			&rec.TargetEventID,
			&rec.TargetEventName,
			&rec.TargetCategoryID,
			&rec.Category,
			&rec.Subject,
			&rec.Headline,
			&rec.BodyText,
			&rec.CTALabel,
			&rec.CTAURL,
			&rec.PromoCode,
			&rec.TotalRecipients,
			&rec.SentCount,
			&rec.FailedCount,
			&rec.Status,
			&rec.CreatedBy,
			&rec.CreatedAt,
			&completedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris riwayat broadcast email: %w", err)
		}
		if completedAt.Valid && !completedAt.Time.IsZero() {
			t := completedAt.Time
			rec.CompletedAt = &t
		}
		list = append(list, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi riwayat broadcast email: %w", err)
	}

	return list, nil
}

func (r *broadcastRepository) GetBroadcastByID(ctx context.Context, id string) (*models.EmailBroadcastRecord, error) {
	var rec models.EmailBroadcastRecord
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT b.id, b.target_scope, COALESCE(b.target_event_id, ''), COALESCE(e.title, ''),
		       COALESCE(b.target_category_id, ''), b.category, b.subject, b.headline, b.body_text,
		       COALESCE(b.cta_label, ''), COALESCE(b.cta_url, ''), COALESCE(b.promo_code, ''),
		       b.total_recipients, b.sent_count, b.failed_count, b.status, b.created_by,
		       b.created_at, b.completed_at
		FROM email_broadcasts b
		LEFT JOIN events e ON b.target_event_id = e.id
		WHERE b.id = $1
	`, id).Scan(
		&rec.ID,
		&rec.TargetScope,
		&rec.TargetEventID,
		&rec.TargetEventName,
		&rec.TargetCategoryID,
		&rec.Category,
		&rec.Subject,
		&rec.Headline,
		&rec.BodyText,
		&rec.CTALabel,
		&rec.CTAURL,
		&rec.PromoCode,
		&rec.TotalRecipients,
		&rec.SentCount,
		&rec.FailedCount,
		&rec.Status,
		&rec.CreatedBy,
		&rec.CreatedAt,
		&completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil detail broadcast email: %w", err)
	}
	if completedAt.Valid && !completedAt.Time.IsZero() {
		t := completedAt.Time
		rec.CompletedAt = &t
	}
	return &rec, nil
}

func (r *broadcastRepository) ResolveRecipients(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error) {
	return r.ResolveRecipientsFiltered(ctx, scope, eventID, "", false)
}

func (r *broadcastRepository) ResolveRecipientsFiltered(ctx context.Context, scope, eventID, categoryID string, excludePromoOptOut bool) ([]models.BroadcastRecipient, error) {
	var query string
	var args []interface{}

	optOutClause := ""
	if excludePromoOptOut {
		optOutClause = " AND COALESCE(u.email_promo_opt_out, FALSE) = FALSE "
	}

	switch strings.ToUpper(strings.TrimSpace(scope)) {
	case "ALL_USERS":
		query = fmt.Sprintf(`
			SELECT DISTINCT ON (LOWER(TRIM(u.email)))
			       COALESCE(NULLIF(TRIM(u.name), ''), 'Penikmat Musik'),
			       LOWER(TRIM(u.email)),
			       '',
			       COALESCE(u.unsubscribe_token, '')
			FROM users u
			WHERE TRIM(u.email) <> '' %s
			ORDER BY LOWER(TRIM(u.email)), u.created_at DESC
		`, optOutClause)
	case "EVENT_ATTENDEES":
		query = fmt.Sprintf(`
			SELECT DISTINCT ON (LOWER(TRIM(o.user_email)))
			       COALESCE(NULLIF(TRIM(o.user_name), ''), 'Pemegang Tiket'),
			       LOWER(TRIM(o.user_email)),
			       o.order_code,
			       COALESCE(u.unsubscribe_token, '')
			FROM orders o
			LEFT JOIN users u ON LOWER(TRIM(u.email)) = LOWER(TRIM(o.user_email))
			WHERE o.event_id = $1
			  AND UPPER(o.status) IN ('ISSUED', 'VERIFIED', 'REMINDED')
			  AND TRIM(o.user_email) <> '' %s
			ORDER BY LOWER(TRIM(o.user_email)), o.created_at DESC
		`, optOutClause)
		args = append(args, strings.TrimSpace(eventID))
	case "CATEGORY_ATTENDEES":
		query = fmt.Sprintf(`
			SELECT DISTINCT ON (LOWER(TRIM(o.user_email)))
			       COALESCE(NULLIF(TRIM(o.user_name), ''), 'Pemegang Tiket'),
			       LOWER(TRIM(o.user_email)),
			       o.order_code,
			       COALESCE(u.unsubscribe_token, '')
			FROM orders o
			JOIN ticket_categories tc ON tc.event_id = o.event_id AND LOWER(TRIM(tc.name)) = LOWER(TRIM(o.category_name))
			LEFT JOIN users u ON LOWER(TRIM(u.email)) = LOWER(TRIM(o.user_email))
			WHERE o.event_id = $1
			  AND tc.id = $2
			  AND UPPER(o.status) IN ('ISSUED', 'VERIFIED', 'REMINDED')
			  AND TRIM(o.user_email) <> '' %s
			ORDER BY LOWER(TRIM(o.user_email)), o.created_at DESC
		`, optOutClause)
		args = append(args, strings.TrimSpace(eventID), strings.TrimSpace(categoryID))
	case "ALL_CUSTOMERS":
		optOutCombined := ""
		if excludePromoOptOut {
			optOutCombined = " WHERE COALESCE(is_opted_out, FALSE) = FALSE "
		}
		query = fmt.Sprintf(`
			SELECT DISTINCT ON (email_key) name, email_key, order_code, unsub_token
			FROM (
				SELECT COALESCE(NULLIF(TRIM(name), ''), 'Penikmat Musik') AS name,
				       LOWER(TRIM(email)) AS email_key,
				       '' AS order_code,
				       COALESCE(unsubscribe_token, '') AS unsub_token,
				       COALESCE(email_promo_opt_out, FALSE) AS is_opted_out,
				       created_at
				FROM users
				WHERE TRIM(email) <> ''
				UNION ALL
				SELECT COALESCE(NULLIF(TRIM(o.user_name), ''), 'Pemegang Tiket') AS name,
				       LOWER(TRIM(o.user_email)) AS email_key,
				       o.order_code,
				       COALESCE(u.unsubscribe_token, '') AS unsub_token,
				       COALESCE(u.email_promo_opt_out, FALSE) AS is_opted_out,
				       o.created_at
				FROM orders o
				LEFT JOIN users u ON LOWER(TRIM(u.email)) = LOWER(TRIM(o.user_email))
				WHERE TRIM(o.user_email) <> ''
			) combined
			%s
			ORDER BY email_key, created_at DESC
		`, optOutCombined)
	default:
		return nil, fmt.Errorf("target scope broadcast tidak dikenali: %s", scope)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar penerima broadcast: %w", err)
	}
	defer rows.Close()

	recipients := make([]models.BroadcastRecipient, 0)
	for rows.Next() {
		var rec models.BroadcastRecipient
		if err := rows.Scan(&rec.Name, &rec.Email, &rec.OrderCode, &rec.UnsubscribeToken); err != nil {
			return nil, fmt.Errorf("gagal membaca data penerima broadcast: %w", err)
		}
		if strings.TrimSpace(rec.UnsubscribeToken) == "" {
			newTok := uuid.NewString()
			_, _ = r.db.ExecContext(ctx, "UPDATE users SET unsubscribe_token = $1 WHERE LOWER(TRIM(email)) = LOWER($2) AND COALESCE(unsubscribe_token, '') = ''", newTok, rec.Email)
			rec.UnsubscribeToken = newTok
		}
		recipients = append(recipients, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi penerima broadcast: %w", err)
	}

	return recipients, nil
}

func (r *broadcastRepository) CreateBroadcastRecipients(ctx context.Context, broadcastID string, recipients []models.BroadcastRecipient) error {
	for _, rec := range recipients {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO email_broadcast_recipients (id, broadcast_id, recipient_name, recipient_email, order_code, status)
			VALUES ($1, $2, $3, $4, $5, 'PENDING')
		`, "brcp-"+uuid.NewString()[:8], broadcastID, rec.Name, rec.Email, rec.OrderCode)
		if err != nil {
			return fmt.Errorf("gagal menyimpan data penerima broadcast: %w", err)
		}
	}
	return nil
}

func (r *broadcastRepository) UpdateRecipientStatus(ctx context.Context, broadcastID, email, status, errorMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE email_broadcast_recipients
		SET status = $3,
		    error_message = $4,
		    sent_at = CASE WHEN $3 = 'SENT' THEN NOW() ELSE sent_at END
		WHERE broadcast_id = $1 AND LOWER(recipient_email) = LOWER($2)
	`, broadcastID, email, status, errorMsg)
	if err != nil {
		return fmt.Errorf("gagal memperbarui status penerima broadcast: %w", err)
	}
	return nil
}

func (r *broadcastRepository) GetBroadcastRecipients(ctx context.Context, broadcastID string) ([]models.BroadcastRecipientRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, broadcast_id, recipient_name, recipient_email, COALESCE(order_code, ''), status, COALESCE(error_message, ''), sent_at, created_at
		FROM email_broadcast_recipients
		WHERE broadcast_id = $1
		ORDER BY CASE WHEN status = 'FAILED' THEN 0 ELSE 1 END, created_at ASC
	`, broadcastID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar penerima broadcast: %w", err)
	}
	defer rows.Close()

	list := make([]models.BroadcastRecipientRecord, 0)
	for rows.Next() {
		var rec models.BroadcastRecipientRecord
		var sentAt sql.NullTime
		if err := rows.Scan(&rec.ID, &rec.BroadcastID, &rec.RecipientName, &rec.RecipientEmail, &rec.OrderCode, &rec.Status, &rec.ErrorMessage, &sentAt, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("gagal membaca penerima broadcast: %w", err)
		}
		if sentAt.Valid && !sentAt.Time.IsZero() {
			t := sentAt.Time
			rec.SentAt = &t
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

func (r *broadcastRepository) GetFailedRecipients(ctx context.Context, broadcastID string) ([]models.BroadcastRecipient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.recipient_name, r.recipient_email, COALESCE(r.order_code, ''), COALESCE(u.unsubscribe_token, '')
		FROM email_broadcast_recipients r
		LEFT JOIN users u ON LOWER(TRIM(u.email)) = LOWER(TRIM(r.recipient_email))
		WHERE r.broadcast_id = $1 AND r.status = 'FAILED'
	`, broadcastID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar penerima gagal: %w", err)
	}
	defer rows.Close()

	list := make([]models.BroadcastRecipient, 0)
	for rows.Next() {
		var rec models.BroadcastRecipient
		if err := rows.Scan(&rec.Name, &rec.Email, &rec.OrderCode, &rec.UnsubscribeToken); err != nil {
			return nil, fmt.Errorf("gagal membaca penerima gagal: %w", err)
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

func (r *broadcastRepository) UnsubscribeByToken(ctx context.Context, token string) error {
	cleanToken := strings.TrimSpace(token)
	if cleanToken == "" {
		return ErrNotFound
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET email_promo_opt_out = TRUE,
		    updated_at = NOW()
		WHERE unsubscribe_token = $1
	`, cleanToken)
	if err != nil {
		return fmt.Errorf("gagal memproses berhenti berlangganan promo: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
