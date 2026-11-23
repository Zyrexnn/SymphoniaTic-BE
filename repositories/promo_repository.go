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

type promoRepository struct {
	db *sql.DB
}

func NewPromoRepository(db *sql.DB) PromoRepository {
	return &promoRepository{db: db}
}

const promoSelectColumns = `
	SELECT id, code, COALESCE(description, ''), discount_type, discount_value,
	       COALESCE(min_order_amount, 0), COALESCE(max_discount_amount, 0),
	       is_member_only, COALESCE(is_active, TRUE), quota, remaining_quota,
	       valid_from, valid_until, created_at
	FROM promotions
`

func scanPromotion(scanner rowScanner) (models.Promotion, error) {
	var promo models.Promotion
	err := scanner.Scan(
		&promo.ID, &promo.Code, &promo.Description, &promo.DiscountType, &promo.DiscountVal,
		&promo.MinOrderAmount, &promo.MaxDiscount, &promo.IsMemberOnly, &promo.IsActive,
		&promo.Quota, &promo.RemainingQuota, &promo.ValidFrom, &promo.ValidUntil, &promo.CreatedAt,
	)
	if err == nil {
		used := promo.Quota - promo.RemainingQuota
		if used < 0 {
			used = 0
		}
		promo.UsedCount = used
	}
	return promo, err
}

func (r *promoRepository) GetPromoByCode(ctx context.Context, code string) (*models.Promotion, error) {
	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	if cleanCode == "" {
		return nil, ErrPromoInvalid
	}

	promo, err := scanPromotion(r.db.QueryRowContext(ctx, promoSelectColumns+`
		WHERE UPPER(code) = $1
	`, cleanCode))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromoInvalid
		}
		return nil, fmt.Errorf("gagal mengambil data promo: %w", err)
	}

	return &promo, nil
}

func (r *promoRepository) HasUserUsedPromo(ctx context.Context, promotionID, userID string) (bool, error) {
	if strings.TrimSpace(userID) == "" {
		return false, nil
	}
	var used bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_promo_usages WHERE promotion_id = $1 AND user_id = $2
		)
	`, promotionID, userID).Scan(&used)
	if err != nil {
		return false, fmt.Errorf("gagal memeriksa riwayat pemakaian promo: %w", err)
	}
	return used, nil
}

func (r *promoRepository) GetAllPromos(ctx context.Context) ([]models.Promotion, error) {
	rows, err := r.db.QueryContext(ctx, promoSelectColumns+`
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar promo: %w", err)
	}
	defer rows.Close()

	promos := make([]models.Promotion, 0)
	for rows.Next() {
		promo, err := scanPromotion(rows)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca data promo: %w", err)
		}
		promos = append(promos, promo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi daftar promo: %w", err)
	}
	return promos, nil
}

func (r *promoRepository) GetActivePromos(ctx context.Context) ([]models.Promotion, error) {
	rows, err := r.db.QueryContext(ctx, promoSelectColumns+`
		WHERE COALESCE(is_active, TRUE) = TRUE
		  AND remaining_quota > 0
		  AND NOW() >= valid_from
		  AND NOW() <= valid_until
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil promo aktif: %w", err)
	}
	defer rows.Close()

	promos := make([]models.Promotion, 0)
	for rows.Next() {
		promo, err := scanPromotion(rows)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca data promo aktif: %w", err)
		}
		promos = append(promos, promo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi promo aktif: %w", err)
	}
	return promos, nil
}

func (r *promoRepository) CreatePromo(ctx context.Context, req *models.CreatePromoRequest) (*models.Promotion, error) {
	promoID := "promo-" + uuid.New().String()
	now := time.Now()
	cleanCode := strings.ToUpper(strings.TrimSpace(req.Code))
	discountType := strings.ToUpper(strings.TrimSpace(req.DiscountType))

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO promotions (
			id, code, description, discount_type, discount_value,
			min_order_amount, max_discount_amount, is_member_only, is_active,
			quota, remaining_quota, valid_from, valid_until, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10, $11, $12, $13)
	`, promoID, cleanCode, strings.TrimSpace(req.Description), discountType, req.DiscountVal,
		req.MinOrderAmount, req.MaxDiscount, req.IsMemberOnly, req.IsActive,
		req.Quota, req.ValidFrom, req.ValidUntil, now)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrPromoDuplicateCode
		}
		return nil, fmt.Errorf("gagal menyimpan kode promo baru: %w", err)
	}

	return &models.Promotion{
		ID:             promoID,
		Code:           cleanCode,
		Description:    strings.TrimSpace(req.Description),
		DiscountType:   discountType,
		DiscountVal:    req.DiscountVal,
		MinOrderAmount: req.MinOrderAmount,
		MaxDiscount:    req.MaxDiscount,
		IsMemberOnly:   req.IsMemberOnly,
		IsActive:       req.IsActive,
		Quota:          req.Quota,
		RemainingQuota: req.Quota,
		UsedCount:      0,
		ValidFrom:      req.ValidFrom,
		ValidUntil:     req.ValidUntil,
		CreatedAt:      now,
	}, nil
}

func (r *promoRepository) UpdatePromo(ctx context.Context, id string, req *models.UpdatePromoRequest) (*models.Promotion, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi pembaruan promo: %w", err)
	}
	defer tx.Rollback()

	var oldQuota, oldRemaining int
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT quota, remaining_quota, created_at
		FROM promotions
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&oldQuota, &oldRemaining, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data promo untuk diperbarui: %w", err)
	}

	usedCount := oldQuota - oldRemaining
	if usedCount < 0 {
		usedCount = 0
	}
	newRemaining := req.Quota - usedCount
	if newRemaining < 0 {
		newRemaining = 0
	}

	cleanCode := strings.ToUpper(strings.TrimSpace(req.Code))
	discountType := strings.ToUpper(strings.TrimSpace(req.DiscountType))

	_, err = tx.ExecContext(ctx, `
		UPDATE promotions
		SET code = $1, description = $2, discount_type = $3, discount_value = $4,
		    min_order_amount = $5, max_discount_amount = $6, is_member_only = $7,
		    is_active = $8, quota = $9, remaining_quota = $10, valid_from = $11, valid_until = $12
		WHERE id = $13
	`, cleanCode, strings.TrimSpace(req.Description), discountType, req.DiscountVal,
		req.MinOrderAmount, req.MaxDiscount, req.IsMemberOnly,
		req.IsActive, req.Quota, newRemaining, req.ValidFrom, req.ValidUntil, id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrPromoDuplicateCode
		}
		return nil, fmt.Errorf("gagal memperbarui kode promo: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal konfirmasi pembaruan kode promo: %w", err)
	}

	return &models.Promotion{
		ID:             id,
		Code:           cleanCode,
		Description:    strings.TrimSpace(req.Description),
		DiscountType:   discountType,
		DiscountVal:    req.DiscountVal,
		MinOrderAmount: req.MinOrderAmount,
		MaxDiscount:    req.MaxDiscount,
		IsMemberOnly:   req.IsMemberOnly,
		IsActive:       req.IsActive,
		Quota:          req.Quota,
		RemainingQuota: newRemaining,
		UsedCount:      usedCount,
		ValidFrom:      req.ValidFrom,
		ValidUntil:     req.ValidUntil,
		CreatedAt:      createdAt,
	}, nil
}

func (r *promoRepository) TogglePromoActive(ctx context.Context, id string) (bool, error) {
	var isActive bool
	err := r.db.QueryRowContext(ctx, `
		UPDATE promotions
		SET is_active = NOT COALESCE(is_active, TRUE)
		WHERE id = $1
		RETURNING is_active
	`, id).Scan(&isActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, fmt.Errorf("gagal mengubah status aktif kode promo: %w", err)
	}
	return isActive, nil
}

func (r *promoRepository) DeletePromo(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM promotions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("gagal menghapus kode promo: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
