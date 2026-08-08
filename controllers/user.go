package controllers

import (
	"database/sql"
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

// GetUserOrders godoc
// @Summary Mengambil riwayat pesanan pengguna
// @Description Mengambil semua riwayat tiket/pesanan milik user yang sedang login.
// @Tags User Profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param status query string false "Filter status pesanan (ISSUED, CHECKED_IN, REFUNDED, dll)"
// @Success 200 {object} models.APIResponse{data=[]models.OrderRecord} "Berhasil mengambil riwayat pesanan"
// @Failure 401 {object} models.APIResponse "Sesi tidak valid"
// @Router /user/orders [get]
func GetUserOrders(ctx *fiber.Ctx) error {
	authenticatedUserID, _ := ctx.Locals("user_id").(string)
	authenticatedUserEmail, _ := ctx.Locals("user_email").(string)

	if authenticatedUserID == "" && authenticatedUserEmail == "" {
		return utils.ResponseUnauthorized(ctx, "Sesi tidak valid.")
	}

	orderStatusFilter := strings.TrimSpace(ctx.Query("status"))

	baseQuery := `
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2))
	`
	queryArgs := []interface{}{authenticatedUserID, authenticatedUserEmail}

	if orderStatusFilter != "" {
		baseQuery += " AND UPPER(status) = UPPER($3)"
		queryArgs = append(queryArgs, orderStatusFilter)
	}

	baseQuery += " ORDER BY created_at DESC"

	orderRows, err := database.DB.Query(baseQuery, queryArgs...)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil riwayat pesanan.", err)
	}
	defer orderRows.Close()

	userOrders := make([]models.OrderRecord, 0)
	for orderRows.Next() {
		var orderRecord models.OrderRecord
		rowScanErr := orderRows.Scan(
			&orderRecord.ID, &orderRecord.OrderCode, &orderRecord.UserID, &orderRecord.EventID, &orderRecord.EventTitle, &orderRecord.Artist, &orderRecord.Venue, &orderRecord.Date,
			&orderRecord.CategoryName, &orderRecord.Quantity, &orderRecord.TotalPrice, &orderRecord.UserName, &orderRecord.UserEmail, &orderRecord.QRCode, &orderRecord.Status,
			&orderRecord.PaymentMethod, &orderRecord.CreatedAt,
		)
		if rowScanErr != nil {
			continue
		}
		userOrders = append(userOrders, orderRecord)
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil riwayat pesanan.", userOrders)
}

// GetUserOrderByCode godoc
// @Summary Mengambil detail 1 pesanan pengguna
// @Description Mengambil detail spesifik pesanan berdasarkan kode pesanan untuk user yang login.
// @Tags User Profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param orderCode path string true "Order Code"
// @Success 200 {object} models.APIResponse{data=models.OrderRecord} "Berhasil mengambil detail pesanan"
// @Failure 404 {object} models.APIResponse "Pesanan tidak ditemukan"
// @Router /user/orders/{orderCode} [get]
func GetUserOrderByCode(ctx *fiber.Ctx) error {
	authenticatedUserID, _ := ctx.Locals("user_id").(string)
	authenticatedUserEmail, _ := ctx.Locals("user_email").(string)
	requestedOrderCode := ctx.Params("orderCode")

	if requestedOrderCode == "" {
		return utils.ResponseBadRequest(ctx, "Kode pesanan wajib diisi.")
	}

	var orderRecord models.OrderRecord
	err := database.DB.QueryRow(`
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE LOWER(order_code) = LOWER($1) AND (user_id = $2 OR LOWER(user_email) = LOWER($3))
	`, requestedOrderCode, authenticatedUserID, authenticatedUserEmail).Scan(
		&orderRecord.ID, &orderRecord.OrderCode, &orderRecord.UserID, &orderRecord.EventID, &orderRecord.EventTitle, &orderRecord.Artist, &orderRecord.Venue, &orderRecord.Date,
		&orderRecord.CategoryName, &orderRecord.Quantity, &orderRecord.TotalPrice, &orderRecord.UserName, &orderRecord.UserEmail, &orderRecord.QRCode, &orderRecord.Status,
		&orderRecord.PaymentMethod, &orderRecord.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ResponseNotFound(ctx, "Pesanan tidak ditemukan atau bukan milik akun Anda.")
		}
		return utils.ResponseInternalError(ctx, "Gagal mengambil detail pesanan.", err)
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil detail pesanan.", orderRecord)
}

// UpdateUserProfile godoc
// @Summary Memperbarui profil pengguna
// @Description Mengubah nama lengkap dan nomor telepon pengguna terautentikasi.
// @Tags User Profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body models.UpdateProfileInput true "Update Profile Payload"
// @Success 200 {object} models.APIResponse{data=models.UserRecord} "Profil berhasil diperbarui"
// @Failure 400 {object} models.APIResponse "Nama wajib diisi"
// @Router /user/profile [put]
func UpdateUserProfile(ctx *fiber.Ctx) error {
	authenticatedUserID, _ := ctx.Locals("user_id").(string)
	if authenticatedUserID == "" {
		return utils.ResponseUnauthorized(ctx, "Sesi tidak valid.")
	}

	var updateProfileReq models.UpdateProfileInput
	if err := ctx.BodyParser(&updateProfileReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	updateProfileReq.Name = strings.TrimSpace(updateProfileReq.Name)
	updateProfileReq.Phone = strings.TrimSpace(updateProfileReq.Phone)

	if updateProfileReq.Name == "" {
		return utils.ResponseBadRequest(ctx, "Nama lengkap tidak boleh kosong.")
	}

	updateResult, err := database.DB.Exec(`
		UPDATE users SET name = $1, phone = $2, updated_at = NOW() WHERE id = $3
	`, updateProfileReq.Name, updateProfileReq.Phone, authenticatedUserID)

	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal memperbarui profil pengguna.", err)
	}

	affectedRows, _ := updateResult.RowsAffected()
	if affectedRows == 0 {
		return utils.ResponseNotFound(ctx, "Pengguna tidak ditemukan.")
	}

	var updatedUserProfile models.UserRecord
	_ = database.DB.QueryRow(`
		SELECT id, email, name, phone, role, is_verified, created_at, updated_at
		FROM users WHERE id = $1
	`, authenticatedUserID).Scan(
		&updatedUserProfile.ID, &updatedUserProfile.Email, &updatedUserProfile.Name, &updatedUserProfile.Phone,
		&updatedUserProfile.Role, &updatedUserProfile.IsVerified, &updatedUserProfile.CreatedAt, &updatedUserProfile.UpdatedAt,
	)

	return utils.ResponseOK(ctx, "Profil berhasil diperbarui.", updatedUserProfile)
}

// ChangeUserPassword godoc
// @Summary Mengubah kata sandi akun
// @Description Mengubah kata sandi user dengan mengonfirmasi kata sandi lama.
// @Tags User Profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body models.ChangePasswordInput true "Change Password Payload"
// @Success 200 {object} models.APIResponse "Kata sandi akun Anda berhasil diperbarui"
// @Failure 400 {object} models.APIResponse "Kata sandi lama salah / minimal 6 karakter"
// @Router /user/change-password [post]
func ChangeUserPassword(ctx *fiber.Ctx) error {
	authenticatedUserID, _ := ctx.Locals("user_id").(string)
	if authenticatedUserID == "" {
		return utils.ResponseUnauthorized(ctx, "Sesi tidak valid.")
	}

	var changePasswordReq models.ChangePasswordInput
	if err := ctx.BodyParser(&changePasswordReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	if changePasswordReq.OldPassword == "" || changePasswordReq.NewPassword == "" {
		return utils.ResponseBadRequest(ctx, "Kata sandi lama dan baru wajib diisi.")
	}

	if len(changePasswordReq.NewPassword) < 6 {
		return utils.ResponseBadRequest(ctx, "Kata sandi baru minimal 6 karakter.")
	}

	var currentPasswordHash string
	err := database.DB.QueryRow("SELECT password_hash FROM users WHERE id = $1", authenticatedUserID).Scan(&currentPasswordHash)
	if err != nil {
		return utils.ResponseNotFound(ctx, "Data akun tidak ditemukan.")
	}

	if !utils.CheckPasswordHash(changePasswordReq.OldPassword, currentPasswordHash) {
		return utils.ResponseBadRequest(ctx, "Kata sandi lama Anda tidak sesuai.")
	}

	newPasswordHash, err := utils.HashPassword(changePasswordReq.NewPassword)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengamankan kata sandi baru.", err)
	}

	_, err = database.DB.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", newPasswordHash, authenticatedUserID)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal memperbarui kata sandi.", err)
	}

	return utils.ResponseOK(ctx, "Kata sandi akun Anda berhasil diperbarui.", nil)
}

// GetUserDashboardSummary godoc
// @Summary Mengambil ringkasan statistik dashboard user
// @Description Mengambil statistik total tiket dibeli, konser mendatang, konser selesai, dan refund aktif.
// @Tags User Profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.UserDashboardSummary} "Berhasil mengambil ringkasan statistik akun"
// @Router /user/dashboard-summary [get]
func GetUserDashboardSummary(ctx *fiber.Ctx) error {
	authenticatedUserID, _ := ctx.Locals("user_id").(string)
	authenticatedUserEmail, _ := ctx.Locals("user_email").(string)

	var dashboardSummary models.UserDashboardSummary

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
	`, authenticatedUserID, authenticatedUserEmail).Scan(
		&dashboardSummary.TotalTicketsBought,
		&dashboardSummary.UpcomingEventsCount,
		&dashboardSummary.PastEventsCount,
		&dashboardSummary.ActiveRefundsCount,
	)
	if err != nil && err != sql.ErrNoRows {
		return utils.ResponseInternalError(ctx, "Gagal mengambil ringkasan statistik akun.", err)
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil ringkasan statistik akun.", dashboardSummary)
}

// GetUserRefunds godoc
// @Summary Mengambil daftar pengajuan refund pengguna
// @Description Mengambil riwayat pengajuan refund tiket beserta status persetujuan dari admin.
// @Tags User Profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.RefundRequestRecord} "Berhasil mengambil data pengajuan refund"
// @Router /user/refunds [get]
func GetUserRefunds(ctx *fiber.Ctx) error {
	authenticatedUserEmail, _ := ctx.Locals("user_email").(string)
	if authenticatedUserEmail == "" {
		return utils.ResponseUnauthorized(ctx, "Sesi tidak valid.")
	}

	refundRows, err := database.DB.Query(`
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, 
		       r.account_holder, COALESCE(r.reason, ''), r.refund_amount, r.status, 
		       COALESCE(r.admin_note, ''), r.created_at, r.updated_at,
		       COALESCE(o.event_title, ''), COALESCE(o.category_name, ''), 
		       COALESCE(o.quantity, 0), COALESCE(o.user_name, '')
		FROM refund_requests r
		LEFT JOIN orders o ON r.order_id = o.id
		WHERE LOWER(r.user_email) = LOWER($1)
		ORDER BY r.created_at DESC
	`, authenticatedUserEmail)

	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil data pengajuan refund.", err)
	}
	defer refundRows.Close()

	refundList := make([]models.RefundRequestRecord, 0)
	for refundRows.Next() {
		var refundRecord models.RefundRequestRecord
		rowScanErr := refundRows.Scan(
			&refundRecord.ID, &refundRecord.OrderID, &refundRecord.OrderCode, &refundRecord.UserEmail, &refundRecord.BankName, &refundRecord.AccountNumber,
			&refundRecord.AccountHolder, &refundRecord.Reason, &refundRecord.RefundAmount, &refundRecord.Status, &refundRecord.AdminNote,
			&refundRecord.CreatedAt, &refundRecord.UpdatedAt, &refundRecord.EventTitle, &refundRecord.CategoryName, &refundRecord.Quantity, &refundRecord.UserName,
		)
		if rowScanErr != nil {
			continue
		}
		refundList = append(refundList, refundRecord)
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil data pengajuan refund.", refundList)
}
