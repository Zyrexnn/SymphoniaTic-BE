package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

	var evtTitle, evtArtist, evtVenue, evtDate, evtTime, evtOpenGate, evtAddress string
	var evtClosed bool
	var rawEventDateTime sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT title, artist, venue, date, time, COALESCE(open_gate, ''), COALESCE(address, ''), is_closed, event_date_time
		FROM events
		WHERE id = $1
	`, req.EventID).Scan(&evtTitle, &evtArtist, &evtVenue, &evtDate, &evtTime, &evtOpenGate, &evtAddress, &evtClosed, &rawEventDateTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data konser: %w", err)
	}

	if evtClosed {
		return nil, ErrEventClosed
	}

	now := time.Now()
	if rawEventDateTime.Valid && !rawEventDateTime.Time.IsZero() {
		if now.After(rawEventDateTime.Time) {
			return nil, ErrEventClosed
		}
	} else if dt, parseErr := utils.ParseConcertDateTime(evtDate, evtTime); parseErr == nil && !dt.IsZero() {
		if now.After(dt) {
			return nil, ErrEventClosed
		}
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

	normalizedSeats := make([]string, 0, len(req.SeatNumbers))
	seenSeats := make(map[string]bool, len(req.SeatNumbers))
	for _, rawCode := range req.SeatNumbers {
		code := strings.ToUpper(strings.TrimSpace(rawCode))
		if code != "" && !seenSeats[code] {
			seenSeats[code] = true
			normalizedSeats = append(normalizedSeats, code)
		}
	}

	if len(req.SeatNumbers) > 0 && len(normalizedSeats) != req.Quantity {
		return nil, fmt.Errorf("%w (jumlah kursi dipilih %d, dibutuhkan %d)", ErrInvalidSeatSelection, len(normalizedSeats), req.Quantity)
	}

	var assignedSeatIDs []string
	var assignedSeatCodes []string

	if len(normalizedSeats) > 0 {
		seatRows, err := tx.QueryContext(ctx, `
			SELECT id, seat_code, category_id, status
			FROM seats
			WHERE event_id = $1 AND seat_code = ANY($2)
			FOR UPDATE
		`, eventID, pq.Array(normalizedSeats))
		if err != nil {
			return nil, fmt.Errorf("gagal memverifikasi ketersediaan kursi: %w", err)
		}

		type lockedSeat struct {
			id         string
			seatCode   string
			categoryID string
			status     string
		}
		foundByCode := make(map[string]lockedSeat, len(normalizedSeats))
		for seatRows.Next() {
			var ls lockedSeat
			if err := seatRows.Scan(&ls.id, &ls.seatCode, &ls.categoryID, &ls.status); err != nil {
				seatRows.Close()
				return nil, fmt.Errorf("gagal membaca status kursi: %w", err)
			}
			foundByCode[strings.ToUpper(ls.seatCode)] = ls
		}
		if err := seatRows.Err(); err != nil {
			seatRows.Close()
			return nil, fmt.Errorf("kesalahan membaca data kursi: %w", err)
		}
		seatRows.Close()

		if len(foundByCode) != len(normalizedSeats) {
			return nil, fmt.Errorf("%w (nomor kursi tidak terdaftar pada konser ini)", ErrInvalidSeatSelection)
		}

		for _, code := range normalizedSeats {
			ls := foundByCode[code]
			if ls.categoryID != catID {
				return nil, fmt.Errorf("%w (kursi %s tidak termasuk dalam kategori %s)", ErrInvalidSeatSelection, ls.seatCode, catName)
			}
			if strings.ToUpper(ls.status) != "AVAILABLE" {
				return nil, fmt.Errorf("%w: kursi %s", ErrSeatUnavailable, ls.seatCode)
			}
			assignedSeatIDs = append(assignedSeatIDs, ls.id)
			assignedSeatCodes = append(assignedSeatCodes, ls.seatCode)
		}
	} else {
		seatRows, err := tx.QueryContext(ctx, `
			SELECT id, seat_code
			FROM seats
			WHERE event_id = $1 AND category_id = $2 AND status = 'AVAILABLE'
			ORDER BY row_label ASC, seat_number ASC
			LIMIT $3
			FOR UPDATE
		`, eventID, catID, req.Quantity)
		if err != nil {
			return nil, fmt.Errorf("gagal mengalokasikan kursi otomatis: %w", err)
		}
		for seatRows.Next() {
			var sid, scode string
			if err := seatRows.Scan(&sid, &scode); err != nil {
				seatRows.Close()
				return nil, fmt.Errorf("gagal membaca kursi otomatis: %w", err)
			}
			assignedSeatIDs = append(assignedSeatIDs, sid)
			assignedSeatCodes = append(assignedSeatCodes, scode)
		}
		if err := seatRows.Err(); err != nil {
			seatRows.Close()
			return nil, fmt.Errorf("kesalahan iterasi alokasi kursi otomatis: %w", err)
		}
		seatRows.Close()

		if len(assignedSeatIDs) > 0 && len(assignedSeatIDs) < req.Quantity {
			return nil, fmt.Errorf("%w (kursi fisik tersedia hanya %d)", ErrSeatUnavailable, len(assignedSeatIDs))
		}
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
	seatNumbersStr := strings.Join(assignedSeatCodes, ", ")
	resolvedAttendees := models.ResolveAttendeeNames(req.AttendeeNames, req.UserName, req.Quantity)
	attendeeNamesJSON := models.EncodeAttendeeNames(resolvedAttendees)
	totalPrice := price * float64(req.Quantity)
	dateFull := fmt.Sprintf("%s @ %s", evtDate, evtTime)
	now = time.Now()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, order_code, user_id, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, seat_numbers, attendee_names, checked_in_seats, status, payment_method, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, '', 'ISSUED', 'SANDBOX_PAYMENT', $17)
	`, orderID, orderCode, authenticatedUserID, eventID, evtTitle, evtArtist, evtVenue, dateFull, catName, req.Quantity, totalPrice, req.UserName, req.UserEmail, qrCode, seatNumbersStr, attendeeNamesJSON, now)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat data pesanan tiket: %w", err)
	}

	if len(assignedSeatIDs) > 0 {
		_, err = tx.ExecContext(ctx, `
			UPDATE seats
			SET status = 'BOOKED', order_id = $1
			WHERE id = ANY($2)
		`, orderID, pq.Array(assignedSeatIDs))
		if err != nil {
			return nil, fmt.Errorf("gagal menandai kursi terpesan: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal konfirmasi transaksi pesanan tiket: %w", err)
	}

	orderRecord := &models.OrderRecord{
		ID:            orderID,
		OrderCode:     orderCode,
		UserID:        authenticatedUserID,
		EventID:       eventID,
		EventTitle:    evtTitle,
		Artist:        evtArtist,
		Venue:         evtVenue,
		Date:          dateFull,
		OpenGate:      evtOpenGate,
		EventTime:     evtTime,
		VenueAddress:  evtAddress,
		CategoryName:  catName,
		Quantity:      req.Quantity,
		TotalPrice:    totalPrice,
		UserName:      req.UserName,
		UserEmail:     req.UserEmail,
		QRCode:        qrCode,
		SeatNumbers:   seatNumbersStr,
		AttendeeNames: attendeeNamesJSON,
		Status:        "ISSUED",
		PaymentMethod: "SANDBOX_PAYMENT",
		CreatedAt:     now,
	}
	models.BuildOrderTickets(orderRecord)
	return orderRecord, nil
}

const orderWithEventSelect = `
	SELECT o.id, o.order_code, COALESCE(o.user_id, ''), o.event_id, o.event_title, o.artist, o.venue, o.date,
	       COALESCE(e.open_gate, ''), COALESCE(e.time, ''), COALESCE(e.address, ''),
	       o.category_name, o.quantity, o.total_price, o.user_name, o.user_email, o.qr_code, COALESCE(o.seat_numbers, ''),
	       COALESCE(o.attendee_names, ''), COALESCE(o.checked_in_seats, ''), o.status,
	       COALESCE(o.payment_method, 'SANDBOX_PAYMENT'), o.created_at
	FROM orders o
	LEFT JOIN events e ON o.event_id = e.id
`

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanOrderWithEvent(scanner rowScanner) (models.OrderRecord, error) {
	var record models.OrderRecord
	err := scanner.Scan(
		&record.ID, &record.OrderCode, &record.UserID, &record.EventID, &record.EventTitle,
		&record.Artist, &record.Venue, &record.Date, &record.OpenGate, &record.EventTime, &record.VenueAddress,
		&record.CategoryName, &record.Quantity, &record.TotalPrice, &record.UserName, &record.UserEmail,
		&record.QRCode, &record.SeatNumbers, &record.AttendeeNames, &record.CheckedInSeats, &record.Status,
		&record.PaymentMethod, &record.CreatedAt,
	)
	if err == nil {
		models.BuildOrderTickets(&record)
	}
	return record, err
}

func parseTicketLookupCode(raw string) (cleanCode, baseOrderCode, targetSeat string) {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToUpper(trimmed), "QR-") {
		trimmed = strings.TrimSpace(trimmed[3:])
	}
	cleanCode = trimmed
	if strings.HasPrefix(strings.ToUpper(cleanCode), "SYM-") && len(cleanCode) > 4 {
		rest := cleanCode[4:]
		if idx2 := strings.Index(rest, "-"); idx2 > 0 && idx2+1 < len(rest) {
			baseOrderCode = cleanCode[:4+idx2]
			targetSeat = strings.ToUpper(strings.TrimSpace(rest[idx2+1:]))
		}
	}
	return cleanCode, baseOrderCode, targetSeat
}

func (r *orderRepository) GetOrderByCode(ctx context.Context, code string) (*models.OrderRecord, error) {
	cleanRaw := strings.TrimSpace(code)
	cleanCode, baseOrderCode, _ := parseTicketLookupCode(cleanRaw)

	record, err := scanOrderWithEvent(r.db.QueryRowContext(ctx, orderWithEventSelect+`
		WHERE LOWER(o.order_code) = LOWER($1)
		   OR LOWER(o.order_code) = LOWER($2)
		   OR LOWER(o.qr_code) = LOWER($3)
		   OR ($4 <> '' AND LOWER(o.order_code) = LOWER($4))
	`, cleanRaw, cleanCode, cleanRaw, baseOrderCode))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil data pesanan tiket: %w", err)
	}

	return &record, nil
}

func (r *orderRepository) GetOrderByID(ctx context.Context, id string) (*models.OrderRecord, error) {
	record, err := scanOrderWithEvent(r.db.QueryRowContext(ctx, orderWithEventSelect+`
		WHERE o.id = $1
	`, id))
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
	cleanCode, baseOrderCode, targetSeat := parseTicketLookupCode(orderCode)
	if cleanCode == "" {
		return nil, ErrNotFound
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi check-in tiket: %w", err)
	}
	defer tx.Rollback()

	// Penguncian baris FOR UPDATE OF o memastikan atomisitas status tiket tanpa mengunci baris tabel events yang dapat menimbulkan lock contention
	var record models.OrderRecord
	var evtClosed bool
	var evtOpenGate, evtTime string
	var rawEventDateTime sql.NullTime

	err = tx.QueryRowContext(ctx, `
		SELECT o.id, o.order_code, COALESCE(o.user_id, ''), o.event_id, o.event_title, o.artist, o.venue, o.date,
		       o.category_name, o.quantity, o.total_price, o.user_name, o.user_email, o.qr_code, COALESCE(o.seat_numbers, ''),
		       COALESCE(o.attendee_names, ''), COALESCE(o.checked_in_seats, ''), o.status,
		       COALESCE(o.payment_method, 'SANDBOX_PAYMENT'), o.created_at,
		       e.is_closed, COALESCE(e.open_gate, ''), COALESCE(e.time, ''), e.event_date_time
		FROM orders o
		JOIN events e ON o.event_id = e.id
		WHERE LOWER(o.order_code) = LOWER($1) OR ($2 <> '' AND LOWER(o.order_code) = LOWER($2))
		FOR UPDATE OF o
	`, cleanCode, baseOrderCode).Scan(
		&record.ID, &record.OrderCode, &record.UserID, &record.EventID, &record.EventTitle,
		&record.Artist, &record.Venue, &record.Date,
		&record.CategoryName, &record.Quantity, &record.TotalPrice,
		&record.UserName, &record.UserEmail, &record.QRCode, &record.SeatNumbers,
		&record.AttendeeNames, &record.CheckedInSeats, &record.Status,
		&record.PaymentMethod, &record.CreatedAt,
		&evtClosed, &evtOpenGate, &evtTime, &rawEventDateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal memverifikasi data tiket: %w", err)
	}

	record.OpenGate = evtOpenGate
	record.EventTime = evtTime

	switch strings.ToUpper(strings.TrimSpace(record.Status)) {
	case "CHECKED_IN":
		return nil, ErrTicketAlreadyUsed
	case "REFUNDED", "REFUND_REQUESTED", "CANCELLED":
		return nil, ErrTicketInvalid
	case "ISSUED", "VERIFIED", "REMINDED":
	default:
		return nil, ErrTicketNotIssued
	}

	if evtClosed {
		return nil, ErrEventClosed
	}

	var eventDateTime time.Time
	if rawEventDateTime.Valid && !rawEventDateTime.Time.IsZero() {
		eventDateTime = rawEventDateTime.Time
	} else if dt, parseErr := utils.ParseConcertDateTime(record.Date, evtTime); parseErr == nil && !dt.IsZero() {
		eventDateTime = dt
	}

	var openGateTime time.Time
	if strings.TrimSpace(evtOpenGate) != "" {
		if ogt, parseErr := utils.ParseOpenGateDateTime(record.Date, evtOpenGate); parseErr == nil && !ogt.IsZero() {
			openGateTime = ogt
		} else if !eventDateTime.IsZero() {
			if mins, ok := utils.ParseTimeToMinutes(evtOpenGate); ok {
				y, m, d := eventDateTime.Date()
				openGateTime = time.Date(y, m, d, mins/60, mins%60, 0, 0, eventDateTime.Location())
			}
		}
	}
	if openGateTime.IsZero() && !eventDateTime.IsZero() {
		openGateTime = eventDateTime
	}
	if eventDateTime.IsZero() && !openGateTime.IsZero() {
		eventDateTime = openGateTime
	}

	now := time.Now()
	if !openGateTime.IsZero() && now.Before(openGateTime) {
		return nil, ErrEventNotStarted
	}

	if !eventDateTime.IsZero() && now.After(eventDateTime.Add(6 * time.Hour)) {
		return nil, ErrEventExpired
	}

	isSubSeatScan := targetSeat != "" && !strings.EqualFold(record.OrderCode, cleanCode) && strings.EqualFold(record.OrderCode, baseOrderCode)
	orderSeats := models.ParseCommaList(record.SeatNumbers)
	checkedSeats := models.ParseCommaList(record.CheckedInSeats)

	if isSubSeatScan && len(orderSeats) > 0 {
		matchedSeat := ""
		for _, s := range orderSeats {
			if strings.EqualFold(s, targetSeat) {
				matchedSeat = s
				break
			}
		}
		if matchedSeat == "" {
			return nil, ErrNotFound
		}
		for _, cs := range checkedSeats {
			if strings.EqualFold(cs, matchedSeat) {
				return nil, ErrTicketAlreadyUsed
			}
		}
		checkedSeats = append(checkedSeats, matchedSeat)
		record.CheckedInSeats = strings.Join(checkedSeats, ", ")

		checkedMap := make(map[string]bool, len(checkedSeats))
		for _, cs := range checkedSeats {
			checkedMap[strings.ToUpper(cs)] = true
		}
		allEntered := true
		for _, s := range orderSeats {
			if !checkedMap[strings.ToUpper(s)] {
				allEntered = false
				break
			}
		}

		if allEntered {
			_, err = tx.ExecContext(ctx, "UPDATE orders SET status = 'CHECKED_IN', checked_in_seats = $1 WHERE id = $2", record.CheckedInSeats, record.ID)
			if err == nil {
				record.Status = "CHECKED_IN"
			}
		} else {
			_, err = tx.ExecContext(ctx, "UPDATE orders SET checked_in_seats = $1 WHERE id = $2", record.CheckedInSeats, record.ID)
		}
	} else {
		record.CheckedInSeats = record.SeatNumbers
		_, err = tx.ExecContext(ctx, "UPDATE orders SET status = 'CHECKED_IN', checked_in_seats = $1 WHERE id = $2", record.CheckedInSeats, record.ID)
		if err == nil {
			record.Status = "CHECKED_IN"
		}
	}
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui status check-in tiket: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal menyelesaikan transaksi check-in tiket: %w", err)
	}

	models.BuildOrderTickets(&record)
	return &record, nil
}

func (r *orderRepository) GetUserOrders(ctx context.Context, userID, userEmail, statusFilter string) ([]models.OrderRecord, error) {
	baseQuery := orderWithEventSelect + `
		WHERE (o.user_id = $1 OR LOWER(o.user_email) = LOWER($2))
	`
	queryArgs := []interface{}{userID, userEmail}

	if statusFilter != "" {
		baseQuery += " AND UPPER(o.status) = UPPER($3)"
		queryArgs = append(queryArgs, statusFilter)
	}

	baseQuery += " ORDER BY o.created_at DESC"

	rows, err := r.db.QueryContext(ctx, baseQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat pesanan: %w", err)
	}
	defer rows.Close()

	userOrders := make([]models.OrderRecord, 0)
	for rows.Next() {
		record, err := scanOrderWithEvent(rows)
		if err != nil {
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
	record, err := scanOrderWithEvent(r.db.QueryRowContext(ctx, orderWithEventSelect+`
		WHERE LOWER(o.order_code) = LOWER($1) AND (o.user_id = $2 OR LOWER(o.user_email) = LOWER($3))
	`, code, userID, userEmail))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil detail pesanan pengguna: %w", err)
	}

	return &record, nil
}

func (r *orderRepository) GetAllOrders(ctx context.Context, searchQuery, statusFilter string) ([]models.OrderRecord, error) {
	query := orderWithEventSelect + `
		WHERE 1=1
	`
	var queryArgs []interface{}
	argIdx := 1

	if searchQuery != "" {
		query += fmt.Sprintf(" AND (LOWER(o.order_code) LIKE $%d OR LOWER(o.user_name) LIKE $%d OR LOWER(o.user_email) LIKE $%d OR LOWER(o.event_title) LIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		queryArgs = append(queryArgs, "%"+strings.ToLower(searchQuery)+"%")
		argIdx++
	}

	if statusFilter != "" {
		query += fmt.Sprintf(" AND o.status = $%d", argIdx)
		queryArgs = append(queryArgs, strings.ToUpper(statusFilter))
		argIdx++
	}

	query += " ORDER BY o.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar pesanan: %w", err)
	}
	defer rows.Close()

	var allOrders []models.OrderRecord
	for rows.Next() {
		record, err := scanOrderWithEvent(rows)
		if err != nil {
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
	normalizedTarget := strings.ToUpper(strings.TrimSpace(status))
	if normalizedTarget == "REMINDER" {
		normalizedTarget = "REMINDED"
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi pembaruan status pesanan: %w", err)
	}
	defer tx.Rollback()

	// Penguncian baris FOR UPDATE mencegah race condition saat transisi status berpotensi mengembalikan kuota tiket
	var currentStatus, eventID, categoryName string
	var quantity int
	err = tx.QueryRowContext(ctx, `
		SELECT status, event_id, category_name, quantity
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&currentStatus, &eventID, &categoryName, &quantity)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("gagal mengambil data pesanan untuk pembaruan status: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE orders SET status = $1 WHERE id = $2", normalizedTarget, id)
	if err != nil {
		return fmt.Errorf("gagal memperbarui status pesanan: %w", err)
	}

	normalizedCurrent := strings.ToUpper(strings.TrimSpace(currentStatus))
	isFromActive := normalizedCurrent == "ISSUED" || normalizedCurrent == "VERIFIED" || normalizedCurrent == "REMINDED"
	isToInactive := normalizedTarget == "CANCELLED" || normalizedTarget == "REFUNDED"

	if isFromActive && isToInactive && quantity > 0 {
		_, err = tx.ExecContext(ctx, `
			UPDATE ticket_categories
			SET remaining_quota = remaining_quota + $1
			WHERE event_id = $2 AND name = $3
		`, quantity, eventID, categoryName)
		if err != nil {
			return fmt.Errorf("gagal mengembalikan kuota kategori tiket: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE seats
			SET status = 'AVAILABLE', order_id = NULL
			WHERE order_id = $1
		`, id)
		if err != nil {
			return fmt.Errorf("gagal melepas nomor kursi pesanan: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal konfirmasi transaksi pembaruan status pesanan: %w", err)
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
		WHERE status IN ('ISSUED', 'VERIFIED', 'REMINDED', 'CHECKED_IN')
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
		LEFT JOIN orders o ON e.id = o.event_id AND o.status IN ('ISSUED', 'VERIFIED', 'REMINDED', 'CHECKED_IN')
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
		WHERE status IN ('ISSUED', 'VERIFIED', 'REMINDED', 'CHECKED_IN')
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
		WHERE status IN ('ISSUED', 'VERIFIED', 'REMINDED', 'CHECKED_IN')
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
			COALESCE((SELECT SUM(quantity) FROM orders WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2)) AND UPPER(status) NOT IN ('REFUNDED', 'CANCELLED')), 0),
			COALESCE((SELECT COUNT(*) FROM orders WHERE (user_id = $1 OR LOWER(user_email) = LOWER($2)) AND UPPER(status) IN ('ISSUED', 'VERIFIED', 'REMINDED')), 0),
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

func (r *orderRepository) GetAttendanceSummary(ctx context.Context, eventID string) (*models.GateAttendanceSummary, error) {
	var totalIssued int
	var totalCheckedIn int

	cleanEventID := strings.TrimSpace(eventID)
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN UPPER(status) IN ('ISSUED', 'VERIFIED', 'REMINDED', 'CHECKED_IN') THEN quantity ELSE 0 END), 0),
			COALESCE(SUM(
				CASE
					WHEN UPPER(status) = 'CHECKED_IN' THEN quantity
					WHEN COALESCE(TRIM(checked_in_seats), '') <> '' THEN array_length(string_to_array(TRIM(checked_in_seats), ','), 1)
					ELSE 0
				END
			), 0)
		FROM orders
		WHERE ($1 = '' OR event_id = $1)
	`, cleanEventID).Scan(&totalIssued, &totalCheckedIn)
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung ringkasan kehadiran gate: %w", err)
	}

	remaining := totalIssued - totalCheckedIn
	if remaining < 0 {
		remaining = 0
	}

	percentage := 0.0
	if totalIssued > 0 {
		percentage = math.Round((float64(totalCheckedIn)/float64(totalIssued))*1000) / 10
	}

	return &models.GateAttendanceSummary{
		TotalTicketsIssued: totalIssued,
		TotalCheckedIn:     totalCheckedIn,
		RemainingCheckIn:   remaining,
		CheckInPercentage:  percentage,
	}, nil
}
