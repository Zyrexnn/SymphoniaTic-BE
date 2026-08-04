package controllers

import (
	"database/sql"
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

// GET /api/v1/user/orders - Ambil semua riwayat pesanan milik pengguna terautentikasi
func GetUserOrders(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	userEmail, _ := c.Locals("user_email").(string)

	if userID == "" && userEmail == "" {
		return utils.ResponseUnauthorized(c, "Sesi tidak valid.")
	}

	statusFilter := strings.TrimSpace(c.Query("status"))

	query := `
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2))
	`
	args := []interface{}{userID, userEmail}

	if statusFilter != "" {
		query += " AND UPPER(status) = UPPER($3)"
		args = append(args, statusFilter)
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal mengambil riwayat pesanan.", err)
	}
	defer rows.Close()

	orders := make([]models.OrderRecord, 0)
	for rows.Next() {
		var o models.OrderRecord
		err := rows.Scan(
			&o.ID, &o.OrderCode, &o.UserID, &o.EventID, &o.EventTitle, &o.Artist, &o.Venue, &o.Date,
			&o.CategoryName, &o.Quantity, &o.TotalPrice, &o.UserName, &o.UserEmail, &o.QRCode, &o.Status,
			&o.PaymentMethod, &o.CreatedAt,
		)
		if err != nil {
			continue
		}
		orders = append(orders, o)
	}

	return utils.ResponseOK(c, "Berhasil mengambil riwayat pesanan.", orders)
}

// GET /api/v1/user/orders/:orderCode - Ambil detail 1 pesanan milik pengguna
func GetUserOrderByCode(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	userEmail, _ := c.Locals("user_email").(string)
	orderCode := c.Params("orderCode")

	if orderCode == "" {
		return utils.ResponseBadRequest(c, "Kode pesanan wajib diisi.")
	}

	var o models.OrderRecord
	err := database.DB.QueryRow(`
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE LOWER(order_code) = LOWER($1) AND (user_id = $2 OR LOWER(user_email) = LOWER($3))
	`, orderCode, userID, userEmail).Scan(
		&o.ID, &o.OrderCode, &o.UserID, &o.EventID, &o.EventTitle, &o.Artist, &o.Venue, &o.Date,
		&o.CategoryName, &o.Quantity, &o.TotalPrice, &o.UserName, &o.UserEmail, &o.QRCode, &o.Status,
		&o.PaymentMethod, &o.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ResponseNotFound(c, "Pesanan tidak ditemukan atau bukan milik akun Anda.")
		}
		return utils.ResponseInternalError(c, "Gagal mengambil detail pesanan.", err)
	}

	return utils.ResponseOK(c, "Berhasil mengambil detail pesanan.", o)
}

// PUT /api/v1/user/profile - Update Nama Lengkap & Nomor HP
func UpdateUserProfile(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return utils.ResponseUnauthorized(c, "Sesi tidak valid.")
	}

	var req models.UpdateProfileInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.Name == "" {
		return utils.ResponseBadRequest(c, "Nama lengkap tidak boleh kosong.")
	}

	result, err := database.DB.Exec(`
		UPDATE users SET name = $1, phone = $2, updated_at = NOW() WHERE id = $3
	`, req.Name, req.Phone, userID)

	if err != nil {
		return utils.ResponseInternalError(c, "Gagal memperbarui profil pengguna.", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return utils.ResponseNotFound(c, "Pengguna tidak ditemukan.")
	}

	var updatedUser models.UserRecord
	_ = database.DB.QueryRow(`
		SELECT id, email, name, phone, role, is_verified, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(
		&updatedUser.ID, &updatedUser.Email, &updatedUser.Name, &updatedUser.Phone,
		&updatedUser.Role, &updatedUser.IsVerified, &updatedUser.CreatedAt, &updatedUser.UpdatedAt,
	)

	return utils.ResponseOK(c, "Profil berhasil diperbarui.", updatedUser)
}

// POST /api/v1/user/change-password - Ubah kata sandi dari dalam akun
func ChangeUserPassword(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return utils.ResponseUnauthorized(c, "Sesi tidak valid.")
	}

	var req models.ChangePasswordInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		return utils.ResponseBadRequest(c, "Kata sandi lama dan baru wajib diisi.")
	}

	if len(req.NewPassword) < 6 {
		return utils.ResponseBadRequest(c, "Kata sandi baru minimal 6 karakter.")
	}

	var currentHash string
	err := database.DB.QueryRow("SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		return utils.ResponseNotFound(c, "Data akun tidak ditemukan.")
	}

	if !utils.CheckPasswordHash(req.OldPassword, currentHash) {
		return utils.ResponseBadRequest(c, "Kata sandi lama Anda tidak sesuai.")
	}

	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal mengamankan kata sandi baru.", err)
	}

	_, err = database.DB.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", newHash, userID)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal memperbarui kata sandi.", err)
	}

	return utils.ResponseOK(c, "Kata sandi akun Anda berhasil diperbarui.", nil)
}

// GET /api/v1/user/dashboard-summary - Ringkasan statistik akun user
func GetUserDashboardSummary(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	userEmail, _ := c.Locals("user_email").(string)

	var summary models.UserDashboardSummary

	err := database.DB.QueryRow(`
		WITH user_orders AS (
			SELECT status, quantity
			FROM orders
			WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2))
		),
		user_refunds AS (
			SELECT status
			FROM refund_requests
			WHERE LOWER(user_email) = LOWER($2)
		)
		SELECT
			COALESCE(SUM(CASE WHEN UPPER(status) != 'REFUNDED' THEN quantity ELSE 0 END), 0),
			COALESCE(COUNT(CASE WHEN UPPER(status) = 'ISSUED' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN UPPER(status) = 'CHECKED_IN' THEN 1 END), 0),
			COALESCE((SELECT COUNT(*) FROM user_refunds WHERE UPPER(status) = 'PENDING'), 0)
		FROM user_orders
	`, userID, userEmail).Scan(
		&summary.TotalTicketsBought,
		&summary.UpcomingEventsCount,
		&summary.PastEventsCount,
		&summary.ActiveRefundsCount,
	)
	if err != nil && err != sql.ErrNoRows {
		return utils.ResponseInternalError(c, "Gagal mengambil ringkasan statistik akun.", err)
	}

	return utils.ResponseOK(c, "Berhasil mengambil ringkasan statistik akun.", summary)
}

// GET /api/v1/user/refunds - Monitoring status pengajuan refund user
func GetUserRefunds(c *fiber.Ctx) error {
	userEmail, _ := c.Locals("user_email").(string)
	if userEmail == "" {
		return utils.ResponseUnauthorized(c, "Sesi tidak valid.")
	}

	rows, err := database.DB.Query(`
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
		return utils.ResponseInternalError(c, "Gagal mengambil data pengajuan refund.", err)
	}
	defer rows.Close()

	refunds := make([]models.RefundRequestRecord, 0)
	for rows.Next() {
		var r models.RefundRequestRecord
		err := rows.Scan(
			&r.ID, &r.OrderID, &r.OrderCode, &r.UserEmail, &r.BankName, &r.AccountNumber,
			&r.AccountHolder, &r.Reason, &r.RefundAmount, &r.Status, &r.AdminNote,
			&r.CreatedAt, &r.UpdatedAt, &r.EventTitle, &r.CategoryName, &r.Quantity, &r.UserName,
		)
		if err != nil {
			continue
		}
		refunds = append(refunds, r)
	}

	return utils.ResponseOK(c, "Berhasil mengambil data pengajuan refund.", refunds)
}
