package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/google/uuid"
)

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) CreateOrder(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi basis data: %w", err)
	}
	defer tx.Rollback()

	// Memeriksa status penutupan konser terlebih dahulu sebelum mengunci baris kategori tiket guna meminimalkan durasi dan contention row lock
	if req.EventID == "" && req.TicketCategoryID != "" {
		_ = tx.QueryRowContext(ctx, "SELECT event_id FROM ticket_categories WHERE id = $1", req.TicketCategoryID).Scan(&req.EventID)
	}

	var evtTitle, evtArtist, evtVenue, evtDate, evtTime string
	var evtClosed bool
	err = tx.QueryRowContext(ctx, `
		SELECT title, artist, venue, date, time, is_closed
		FROM events
		WHERE id = $1
	`, req.EventID).Scan(&evtTitle, &evtArtist, &evtVenue, &evtDate, &evtTime, &evtClosed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data konser: %w", err)
	}

	if evtClosed {
		return nil, ErrEventClosed
	}

	if req.UserName == "" && authenticatedUserID != "" {
		_ = tx.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", authenticatedUserID).Scan(&req.UserName)
	}

	// Row locking FOR UPDATE untuk menjamin konsistensi kuota saat persaingan transaksi tinggi (ticket war)
	var catID, eventID, catName string
	var price float64
	var quota, remainingQuota int
	err = tx.QueryRowContext(ctx, `
		SELECT id, event_id, name, price, quota, remaining_quota
		FROM ticket_categories
		WHERE id = $1 AND event_id = $2
		FOR UPDATE
	`, req.TicketCategoryID, req.EventID).Scan(&catID, &eventID, &catName, &price, &quota, &remainingQuota)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal memverifikasi kuota tiket: %w", err)
	}

	if remainingQuota < req.Quantity {
		return nil, fmt.Errorf("%w (sisa kuota: %d)", ErrQuotaExceeded, remainingQuota)
	}

	newRemaining := remainingQuota - req.Quantity
	_, err = tx.ExecContext(ctx, `
		UPDATE ticket_categories
		SET remaining_quota = $1
		WHERE id = $2
	`, newRemaining, catID)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui sisa kuota tiket: %w", err)
	}

	randomNum, err := utils.GenerateCryptoOTP()
	if err != nil {
		randomNum = fmt.Sprintf("%d", time.Now().UnixNano()%900000+100000)
	}
	orderCode := "SYM-" + randomNum
	orderID := uuid.New().String()
	qrCode := fmt.Sprintf("QR-%s", orderCode)
	totalPrice := price * float64(req.Quantity)
	dateFull := fmt.Sprintf("%s @ %s", evtDate, evtTime)
	now := time.Now()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, order_code, user_id, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'ISSUED', 'SANDBOX_PAYMENT', $15)
	`, orderID, orderCode, authenticatedUserID, eventID, evtTitle, evtArtist, evtVenue, dateFull, catName, req.Quantity, totalPrice, req.UserName, req.UserEmail, qrCode, now)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat data pesanan tiket: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal konfirmasi transaksi pesanan tiket: %w", err)
	}

	return &models.OrderRecord{
		ID:            orderID,
		OrderCode:     orderCode,
		UserID:        authenticatedUserID,
		EventID:       eventID,
		EventTitle:    evtTitle,
		Artist:        evtArtist,
		Venue:         evtVenue,
		Date:          dateFull,
		CategoryName:  catName,
		Quantity:      req.Quantity,
		TotalPrice:    totalPrice,
		UserName:      req.UserName,
		UserEmail:     req.UserEmail,
		QRCode:        qrCode,
		Status:        "ISSUED",
		PaymentMethod: "SANDBOX_PAYMENT",
		CreatedAt:     now,
	}, nil
}

func (r *orderRepository) GetOrderByCode(ctx context.Context, code string) (*models.OrderRecord, error) {
	var record models.OrderRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE LOWER(order_code) = LOWER($1)
	`, code).Scan(
		&record.ID, &record.OrderCode, &record.UserID, &record.EventID, &record.EventTitle,
		&record.Artist, &record.Venue, &record.Date, &record.CategoryName, &record.Quantity,
		&record.TotalPrice, &record.UserName, &record.UserEmail, &record.QRCode,
		&record.Status, &record.PaymentMethod, &record.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data pesanan tiket: %w", err)
	}

	return &record, nil
}

func (r *orderRepository) GetOrderByID(ctx context.Context, id string) (*models.OrderRecord, error) {
	var record models.OrderRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE id = $1
	`, id).Scan(
		&record.ID, &record.OrderCode, &record.UserID, &record.EventID, &record.EventTitle,
		&record.Artist, &record.Venue, &record.Date, &record.CategoryName, &record.Quantity,
		&record.TotalPrice, &record.UserName, &record.UserEmail, &record.QRCode,
		&record.Status, &record.PaymentMethod, &record.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data pesanan tiket: %w", err)
	}

	return &record, nil
}

func (r *orderRepository) LookupTicketByCode(ctx context.Context, code string) (*models.OrderRecord, error) {
	return r.GetOrderByCode(ctx, code)
}

func (r *orderRepository) CheckInTicket(ctx context.Context, orderCode string) (*models.OrderRecord, error) {
	cleanCode := strings.TrimSpace(orderCode)
	if strings.HasPrefix(strings.ToUpper(cleanCode), "QR-") {
		cleanCode = cleanCode[3:]
	}
	cleanCode = strings.TrimSpace(cleanCode)
	if cleanCode == "" {
		return nil, ErrNotFound
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi check-in tiket: %w", err)
	}
	defer tx.Rollback()

	// Penguncian baris FOR UPDATE digunakan untuk memastikan tidak terjadi race condition atau double check-in saat pemindaian berulang secara simultan
	var record models.OrderRecord
	err = tx.QueryRowContext(ctx, `
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE LOWER(order_code) = LOWER($1)
		FOR UPDATE
	`, cleanCode).Scan(
		&record.ID, &record.OrderCode, &record.UserID, &record.EventID, &record.EventTitle,
		&record.Artist, &record.Venue, &record.Date, &record.CategoryName, &record.Quantity,
		&record.TotalPrice, &record.UserName, &record.UserEmail, &record.QRCode,
		&record.Status, &record.PaymentMethod, &record.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal memverifikasi data tiket: %w", err)
	}

	switch record.Status {
	case "CHECKED_IN":
		return nil, ErrTicketAlreadyUsed
	case "REFUNDED", "REFUND_REQUESTED", "CANCELLED":
		return nil, ErrTicketInvalid
	case "ISSUED", "VERIFIED":
	default:
		return nil, ErrTicketNotIssued
	}

	_, err = tx.ExecContext(ctx, "UPDATE orders SET status = 'CHECKED_IN' WHERE id = $1", record.ID)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui status check-in tiket: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal menyelesaikan transaksi check-in tiket: %w", err)
	}

	record.Status = "CHECKED_IN"
	return &record, nil
}

func (r *orderRepository) GetUserOrders(ctx context.Context, userID, userEmail, statusFilter string) ([]models.OrderRecord, error) {
	baseQuery := `
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2))
	`
	queryArgs := []interface{}{userID, userEmail}

	if statusFilter != "" {
		baseQuery += " AND UPPER(status) = UPPER($3)"
		queryArgs = append(queryArgs, statusFilter)
	}

	baseQuery += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, baseQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat pesanan: %w", err)
	}
	defer rows.Close()

	userOrders := make([]models.OrderRecord, 0)
	for rows.Next() {
		var record models.OrderRecord
		if err := rows.Scan(
			&record.ID, &record.OrderCode, &record.UserID, &record.EventID, &record.EventTitle,
			&record.Artist, &record.Venue, &record.Date, &record.CategoryName, &record.Quantity,
			&record.TotalPrice, &record.UserName, &record.UserEmail, &record.QRCode,
			&record.Status, &record.PaymentMethod, &record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris pesanan: %w", err)
		}
		userOrders = append(userOrders, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris pesanan: %w", err)
	}

	return userOrders, nil
}

func (r *orderRepository) GetUserOrderByCode(ctx context.Context, code, userID, userEmail string) (*models.OrderRecord, error) {
	var record models.OrderRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE LOWER(order_code) = LOWER($1) AND (user_id = $2 OR LOWER(user_email) = LOWER($3))
	`, code, userID, userEmail).Scan(
		&record.ID, &record.OrderCode, &record.UserID, &record.EventID, &record.EventTitle,
		&record.Artist, &record.Venue, &record.Date, &record.CategoryName, &record.Quantity,
		&record.TotalPrice, &record.UserName, &record.UserEmail, &record.QRCode,
		&record.Status, &record.PaymentMethod, &record.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil detail pesanan pengguna: %w", err)
	}

	return &record, nil
}

func (r *orderRepository) GetAllOrders(ctx context.Context, searchQuery, statusFilter string) ([]models.OrderRecord, error) {
	query := `
		SELECT id, order_code, COALESCE(user_id, ''), event_id, event_title, artist, venue, date, 
		       category_name, quantity, total_price, user_name, user_email, qr_code, status, 
		       COALESCE(payment_method, 'SANDBOX_PAYMENT'), created_at
		FROM orders
		WHERE 1=1
	`
	var queryArgs []interface{}
	argIdx := 1

	if searchQuery != "" {
		query += fmt.Sprintf(" AND (LOWER(order_code) LIKE $%d OR LOWER(user_name) LIKE $%d OR LOWER(user_email) LIKE $%d OR LOWER(event_title) LIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		queryArgs = append(queryArgs, "%"+strings.ToLower(searchQuery)+"%")
		argIdx++
	}

	if statusFilter != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		queryArgs = append(queryArgs, strings.ToUpper(statusFilter))
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar pesanan: %w", err)
	}
	defer rows.Close()

	var allOrders []models.OrderRecord
	for rows.Next() {
		var record models.OrderRecord
		if err := rows.Scan(
			&record.ID, &record.OrderCode, &record.UserID, &record.EventID, &record.EventTitle,
			&record.Artist, &record.Venue, &record.Date, &record.CategoryName, &record.Quantity,
			&record.TotalPrice, &record.UserName, &record.UserEmail, &record.QRCode,
			&record.Status, &record.PaymentMethod, &record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris pesanan: %w", err)
		}
		allOrders = append(allOrders, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris daftar pesanan: %w", err)
	}

	return allOrders, nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, id, status string) error {
	result, err := r.db.ExecContext(ctx, "UPDATE orders SET status = $1 WHERE id = $2", strings.ToUpper(status), id)
	if err != nil {
		return fmt.Errorf("gagal memperbarui status pesanan: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa baris yang diperbarui: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *orderRepository) GetAdminDashboardMetrics(ctx context.Context) (*models.AdminDashboardMetrics, error) {
	metrics := &models.AdminDashboardMetrics{
		EventStats:           make([]models.EventRevenueStat, 0),
		RecentOrders:         make([]models.RecentOrderSummary, 0),
		RevenueTimeline:      make([]models.TimelineStat, 6),
		CategoryDistribution: make([]models.CategoryStat, 0),
	}

	// Agregasi single-pass pada tabel orders memanfaatkan indeks komposit idx_orders_status untuk efisiensi eksekusi
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(total_price), 0),
			COALESCE(SUM(quantity), 0),
			COUNT(*)
		FROM orders
		WHERE status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')
	`).Scan(&metrics.TotalRevenue, &metrics.TicketsSold, &metrics.TotalOrders)
	if err != nil {
		return nil, fmt.Errorf("gagal mengagregasi ringkasan pesanan: %w", err)
	}

	err = r.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(remaining_quota), 0) FROM ticket_categories").Scan(&metrics.RemainingQuota)
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung sisa kuota tiket: %w", err)
	}

	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&metrics.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung total konser: %w", err)
	}

	// Statistik pendapatan per konser
	eventRevenueRows, err := r.db.QueryContext(ctx, `
		SELECT e.id, e.title, COALESCE(SUM(o.total_price), 0) as rev, COALESCE(SUM(o.quantity), 0) as sold
		FROM events e
		LEFT JOIN orders o ON e.id = o.event_id AND o.status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')
		GROUP BY e.id, e.title
		ORDER BY rev DESC
	`)
	if err == nil {
		defer eventRevenueRows.Close()
		for eventRevenueRows.Next() {
			var item models.EventRevenueStat
			if err := eventRevenueRows.Scan(&item.EventID, &item.Title, &item.Revenue, &item.TicketsSold); err == nil {
				metrics.EventStats = append(metrics.EventStats, item)
			}
		}
	}

	// Timeline pendapatan 6 bulan terakhir dengan inisialisasi default 0
	currentTime := time.Now()
	for i := 5; i >= 0; i-- {
		targetMonthTime := currentTime.AddDate(0, -i, 0)
		metrics.RevenueTimeline[5-i] = models.TimelineStat{
			Month:   targetMonthTime.Format("Jan"),
			Revenue: 0,
			Tickets: 0,
		}
	}

	timelineRows, err := r.db.QueryContext(ctx, `
		SELECT 
			TO_CHAR(created_at, 'Mon') as month,
			COALESCE(SUM(total_price), 0) as revenue,
			COALESCE(SUM(quantity), 0) as tickets
		FROM orders
		WHERE status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')
		GROUP BY TO_CHAR(created_at, 'Mon'), DATE_TRUNC('month', created_at)
		ORDER BY DATE_TRUNC('month', created_at)
		LIMIT 6
	`)
	if err == nil {
		defer timelineRows.Close()
		for timelineRows.Next() {
			var item models.TimelineStat
			if err := timelineRows.Scan(&item.Month, &item.Revenue, &item.Tickets); err == nil {
				for idx := range metrics.RevenueTimeline {
					if strings.EqualFold(metrics.RevenueTimeline[idx].Month, item.Month) {
						metrics.RevenueTimeline[idx].Revenue = item.Revenue
						metrics.RevenueTimeline[idx].Tickets = item.Tickets
						break
					}
				}
			}
		}
	}

	// Distribusi kategori tiket terjual
	catRows, err := r.db.QueryContext(ctx, `
		SELECT 
			category_name, 
			COALESCE(SUM(total_price), 0) as total_val
		FROM orders
		WHERE status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')
		GROUP BY category_name
		ORDER BY total_val DESC
	`)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var item models.CategoryStat
			if err := catRows.Scan(&item.Name, &item.Value); err == nil {
				metrics.CategoryDistribution = append(metrics.CategoryDistribution, item)
			}
		}
	}

	// 5 pesanan terbaru
	recentRows, err := r.db.QueryContext(ctx, `
		SELECT id, order_code, event_title, quantity, total_price, user_name, status, created_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT 5
	`)
	if err == nil {
		defer recentRows.Close()
		for recentRows.Next() {
			var item models.RecentOrderSummary
			if err := recentRows.Scan(&item.ID, &item.OrderCode, &item.EventTitle, &item.Quantity, &item.TotalPrice, &item.UserName, &item.Status, &item.CreatedAt); err == nil {
				metrics.RecentOrders = append(metrics.RecentOrders, item)
			}
		}
	}

	return metrics, nil
}

func (r *orderRepository) GetUserDashboardSummary(ctx context.Context, userID, userEmail string) (*models.UserDashboardSummary, error) {
	var summary models.UserDashboardSummary

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE((SELECT SUM(quantity) FROM orders WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2)) AND UPPER(status) != 'REFUNDED'), 0),
			COALESCE((SELECT COUNT(*) FROM orders WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2)) AND UPPER(status) IN ('ISSUED', 'VERIFIED')), 0),
			COALESCE((SELECT COUNT(*) FROM orders WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2)) AND UPPER(status) = 'CHECKED_IN'), 0),
			COALESCE((SELECT COUNT(*) FROM refund_requests WHERE LOWER(user_email) = LOWER($2) AND UPPER(status) = 'PENDING'), 0)
	`, userID, userEmail).Scan(
		&summary.TotalTicketsBought,
		&summary.UpcomingEventsCount,
		&summary.PastEventsCount,
		&summary.ActiveRefundsCount,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("gagal mengambil ringkasan akun pengguna: %w", err)
	}

	return &summary, nil
}
