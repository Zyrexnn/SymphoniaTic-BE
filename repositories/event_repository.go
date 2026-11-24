package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type eventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) GetAllEvents(ctx context.Context) ([]models.EventItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			id, 
			COALESCE(title, ''), 
			COALESCE(artist, ''), 
			COALESCE(venue, ''), 
			COALESCE(date, ''), 
			COALESCE(time, ''), 
			COALESCE(category, 'SIMFONI UTAMA'), 
			COALESCE(category_badge_color, 'bg-blue-900/80 text-blue-200 border-blue-500/40'), 
			COALESCE(image, ''), 
			COALESCE(audio_url, ''), 
			COALESCE(conductor, ''), 
			COALESCE(open_gate, ''), 
			COALESCE(address, ''), 
			COALESCE(organizer, ''), 
			COALESCE(subtitle, ''), 
			COALESCE(rundown, '[]'::jsonb), 
			COALESCE(description, ''), 
			COALESCE(is_closed, FALSE),
			COALESCE(status, 'SCHEDULED'),
			COALESCE(cancellation_reason, ''),
			COALESCE(reschedule_reason, ''),
			COALESCE(original_date, ''),
			COALESCE(original_time, ''),
			status_updated_at,
			event_date_time,
			sales_end_at
		FROM events
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data konser: %w", err)
	}
	defer rows.Close()

	var eventList []models.EventItem
	now := time.Now()
	for rows.Next() {
		var eventItem models.EventItem
		var rundownBytes []byte
		var rawStatusUpdatedAt sql.NullTime
		var rawEventDateTime sql.NullTime
		var rawSalesEndAt sql.NullTime

		if err := rows.Scan(
			&eventItem.ID,
			&eventItem.Title,
			&eventItem.Artist,
			&eventItem.Venue,
			&eventItem.Date,
			&eventItem.Time,
			&eventItem.Category,
			&eventItem.CategoryBadgeColor,
			&eventItem.Image,
			&eventItem.AudioURL,
			&eventItem.Conductor,
			&eventItem.OpenGate,
			&eventItem.Address,
			&eventItem.Organizer,
			&eventItem.Subtitle,
			&rundownBytes,
			&eventItem.Description,
			&eventItem.IsClosed,
			&eventItem.Status,
			&eventItem.CancellationReason,
			&eventItem.RescheduleReason,
			&eventItem.OriginalDate,
			&eventItem.OriginalTime,
			&rawStatusUpdatedAt,
			&rawEventDateTime,
			&rawSalesEndAt,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris data konser: %w", err)
		}
		_ = json.Unmarshal(rundownBytes, &eventItem.Rundown)

		if rawStatusUpdatedAt.Valid && !rawStatusUpdatedAt.Time.IsZero() {
			t := rawStatusUpdatedAt.Time
			eventItem.StatusUpdatedAt = &t
		}

		if rawEventDateTime.Valid && !rawEventDateTime.Time.IsZero() {
			t := rawEventDateTime.Time
			eventItem.EventDateTime = &t
		} else if dt, err := utils.ParseConcertDateTime(eventItem.Date, eventItem.Time); err == nil && !dt.IsZero() {
			eventItem.EventDateTime = &dt
		}
		if rawSalesEndAt.Valid && !rawSalesEndAt.Time.IsZero() {
			t := rawSalesEndAt.Time
			eventItem.SalesEndAt = &t
		}
		eventItem.IsClosed = !eventItem.IsSalesOpen(now)

		eventItem.Categories = []models.TicketCategory{}
		eventList = append(eventList, eventItem)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris konser: %w", err)
	}

	// Pemuatan batch kategori dengan ANY($1) untuk mencegah masalah query N+1 pada daftar konser
	if len(eventList) > 0 {
		eventIDs := make([]string, len(eventList))
		for i, eventItem := range eventList {
			eventIDs[i] = eventItem.ID
		}

		catMap, err := r.GetCategoriesByEventIDs(ctx, eventIDs)
		if err == nil {
			for i := range eventList {
				if cats, exists := catMap[eventList[i].ID]; exists {
					eventList[i].Categories = cats
				}
			}
		}
	}

	return eventList, nil
}

func (r *eventRepository) GetEventByID(ctx context.Context, id string) (*models.EventItem, error) {
	var eventItem models.EventItem
	var rundownBytes []byte

	var rawStatusUpdatedAt sql.NullTime
	var rawEventDateTime sql.NullTime
	var rawSalesEndAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, 
			COALESCE(title, ''), 
			COALESCE(artist, ''), 
			COALESCE(venue, ''), 
			COALESCE(date, ''), 
			COALESCE(time, ''), 
			COALESCE(category, 'SIMFONI UTAMA'), 
			COALESCE(category_badge_color, 'bg-blue-900/80 text-blue-200 border-blue-500/40'), 
			COALESCE(image, ''), 
			COALESCE(audio_url, ''), 
			COALESCE(conductor, ''), 
			COALESCE(open_gate, ''), 
			COALESCE(address, ''), 
			COALESCE(organizer, ''), 
			COALESCE(subtitle, ''), 
			COALESCE(rundown, '[]'::jsonb), 
			COALESCE(description, ''), 
			COALESCE(is_closed, FALSE),
			COALESCE(status, 'SCHEDULED'),
			COALESCE(cancellation_reason, ''),
			COALESCE(reschedule_reason, ''),
			COALESCE(original_date, ''),
			COALESCE(original_time, ''),
			status_updated_at,
			event_date_time,
			sales_end_at
		FROM events
		WHERE id = $1
	`, id).Scan(
		&eventItem.ID,
		&eventItem.Title,
		&eventItem.Artist,
		&eventItem.Venue,
		&eventItem.Date,
		&eventItem.Time,
		&eventItem.Category,
		&eventItem.CategoryBadgeColor,
		&eventItem.Image,
		&eventItem.AudioURL,
		&eventItem.Conductor,
		&eventItem.OpenGate,
		&eventItem.Address,
		&eventItem.Organizer,
		&eventItem.Subtitle,
		&rundownBytes,
		&eventItem.Description,
		&eventItem.IsClosed,
		&eventItem.Status,
		&eventItem.CancellationReason,
		&eventItem.RescheduleReason,
		&eventItem.OriginalDate,
		&eventItem.OriginalTime,
		&rawStatusUpdatedAt,
		&rawEventDateTime,
		&rawSalesEndAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil detail konser: %w", err)
	}

	_ = json.Unmarshal(rundownBytes, &eventItem.Rundown)

	if rawStatusUpdatedAt.Valid && !rawStatusUpdatedAt.Time.IsZero() {
		t := rawStatusUpdatedAt.Time
		eventItem.StatusUpdatedAt = &t
	}

	now := time.Now()
	if rawEventDateTime.Valid && !rawEventDateTime.Time.IsZero() {
		t := rawEventDateTime.Time
		eventItem.EventDateTime = &t
	} else if dt, err := utils.ParseConcertDateTime(eventItem.Date, eventItem.Time); err == nil && !dt.IsZero() {
		eventItem.EventDateTime = &dt
	}
	if rawSalesEndAt.Valid && !rawSalesEndAt.Time.IsZero() {
		t := rawSalesEndAt.Time
		eventItem.SalesEndAt = &t
	}
	eventItem.IsClosed = !eventItem.IsSalesOpen(now)

	categories, err := r.GetCategoriesByEventID(ctx, eventItem.ID)
	if err != nil {
		eventItem.Categories = []models.TicketCategory{}
	} else {
		eventItem.Categories = categories
	}

	return &eventItem, nil
}

func (r *eventRepository) CreateEvent(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error) {
	badgeColor := req.CategoryBadgeColor
	if badgeColor == "" {
		badgeColor = "bg-blue-900/80 text-blue-200 border-blue-500/40"
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	eventID := fmt.Sprintf("evt-%s", uuid.New().String()[:8])
	rundownJSON, _ := json.Marshal(req.Rundown)
	if req.Rundown == nil {
		rundownJSON = []byte("[]")
	}

	var parsedDateTime sql.NullTime
	if dt, err := utils.ParseConcertDateTime(req.Date, req.Time); err == nil && !dt.IsZero() {
		parsedDateTime = sql.NullTime{Time: dt, Valid: true}
	}

	var salesEndNullTime sql.NullTime
	if req.SalesEndAt != nil && !req.SalesEndAt.IsZero() {
		salesEndNullTime = sql.NullTime{Time: *req.SalesEndAt, Valid: true}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO events (id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, conductor, open_gate, address, organizer, subtitle, rundown, description, event_date_time, sales_end_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, eventID, req.Title, req.Artist, req.Venue, req.Date, req.Time, req.Category, badgeColor, req.Image, req.AudioURL, req.Conductor, req.OpenGate, req.Address, req.Organizer, req.Subtitle, string(rundownJSON), req.Description, parsedDateTime, salesEndNullTime)

	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan data konser: %w", err)
	}

	createdCategories := make([]models.TicketCategory, 0, len(req.Categories))
	for idx, catInput := range req.Categories {
		catID := fmt.Sprintf("cat-%s-%d", eventID, idx+1)
		catName := catInput.Name
		if catName == "" {
			catName = fmt.Sprintf("Kategori %d", idx+1)
		}
		catPrice := catInput.Price
		catQuota := catInput.Quota
		if catQuota <= 0 {
			catQuota = 50
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, catID, eventID, catName, catPrice, catQuota, catQuota)

		if err != nil {
			return nil, fmt.Errorf("gagal menyimpan kategori tiket konser: %w", err)
		}

		createdCategories = append(createdCategories, models.TicketCategory{
			ID:             catID,
			EventID:        eventID,
			Name:           catName,
			Price:          catPrice,
			Quota:          catQuota,
			RemainingQuota: catQuota,
			CreatedAt:      time.Now(),
		})
	}

	if len(createdCategories) > 0 {
		generatedSeats := utils.GenerateHallSeats(eventID, createdCategories)
		if err := insertSeatsTx(ctx, tx, generatedSeats); err != nil {
			return nil, fmt.Errorf("gagal membuat daftar nomor kursi konser: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal konfirmasi transaksi pembuatan konser: %w", err)
	}

	createdItem := &models.EventItem{
		ID:                 eventID,
		Title:              req.Title,
		Artist:             req.Artist,
		Venue:              req.Venue,
		Date:               req.Date,
		Time:               req.Time,
		Category:           req.Category,
		CategoryBadgeColor: badgeColor,
		Image:              req.Image,
		AudioURL:           req.AudioURL,
		Conductor:          req.Conductor,
		OpenGate:           req.OpenGate,
		Address:            req.Address,
		Organizer:          req.Organizer,
		Subtitle:           req.Subtitle,
		Description:        req.Description,
		SalesEndAt:         req.SalesEndAt,
		Rundown:            req.Rundown,
		Categories:         createdCategories,
	}
	if parsedDateTime.Valid {
		t := parsedDateTime.Time
		createdItem.EventDateTime = &t
	}
	createdItem.IsClosed = !createdItem.IsSalesOpen(time.Now())
	return createdItem, nil
}

func (r *eventRepository) UpdateEvent(ctx context.Context, id string, req *models.UpdateEventRequest) error {
	rundownJSON, _ := json.Marshal(req.Rundown)
	if req.Rundown == nil {
		rundownJSON = []byte("[]")
	}

	var parsedDateTime sql.NullTime
	if dt, err := utils.ParseConcertDateTime(req.Date, req.Time); err == nil && !dt.IsZero() {
		parsedDateTime = sql.NullTime{Time: dt, Valid: true}
	}

	var salesEndNullTime sql.NullTime
	if req.SalesEndAt != nil && !req.SalesEndAt.IsZero() {
		salesEndNullTime = sql.NullTime{Time: *req.SalesEndAt, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE events
		SET title = $1, artist = $2, venue = $3, date = $4, time = $5, category = $6, 
		    category_badge_color = $7, image = $8, audio_url = COALESCE(NULLIF($9, ''), audio_url), conductor = $10, 
		    open_gate = $11, address = $12, organizer = $13, subtitle = $14, rundown = $15, description = $16,
		    event_date_time = $17, sales_end_at = $18
		WHERE id = $19
	`, req.Title, req.Artist, req.Venue, req.Date, req.Time, req.Category, req.CategoryBadgeColor, req.Image, strings.TrimSpace(req.AudioURL), req.Conductor, req.OpenGate, req.Address, req.Organizer, req.Subtitle, string(rundownJSON), req.Description, parsedDateTime, salesEndNullTime, id)

	if err != nil {
		return fmt.Errorf("gagal memperbarui konser: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa baris yang terpengaruh: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *eventRepository) DeleteEvent(ctx context.Context, id string) error {
	// Cegah error foreign key 500 apabila konser sudah memiliki transaksi tiket aktif
	var orderCount int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders WHERE event_id = $1", id).Scan(&orderCount); err != nil {
		return fmt.Errorf("gagal memeriksa transaksi terkait konser: %w", err)
	}
	if orderCount > 0 {
		return ErrEventHasOrders
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM events WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("gagal menghapus konser: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa baris yang terhapus: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *eventRepository) ToggleEventClose(ctx context.Context, id string) (bool, error) {
	var isClosed bool
	var status string
	err := r.db.QueryRowContext(ctx, "SELECT is_closed, COALESCE(status, 'SCHEDULED') FROM events WHERE id = $1", id).Scan(&isClosed, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrNotFound
		}
		return false, fmt.Errorf("gagal memeriksa status penutupan konser: %w", err)
	}

	if strings.EqualFold(status, "CANCELLED") {
		return false, ErrEventCancelled
	}

	newState := !isClosed
	_, err = r.db.ExecContext(ctx, "UPDATE events SET is_closed = $1 WHERE id = $2", newState, id)
	if err != nil {
		return false, fmt.Errorf("gagal mengubah status penutupan konser: %w", err)
	}

	return newState, nil
}

func (r *eventRepository) CancelEvent(ctx context.Context, id string, reason string) (*models.EventItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi pembatalan konser: %w", err)
	}
	defer tx.Rollback()

	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(status, 'SCHEDULED') FROM events WHERE id = $1 FOR UPDATE", id).Scan(&currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal memeriksa status konser: %w", err)
	}

	if strings.EqualFold(currentStatus, "CANCELLED") {
		return nil, ErrEventAlreadyCancelled
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE events
		SET status = 'CANCELLED',
		    is_closed = TRUE,
		    cancellation_reason = $2,
		    status_updated_at = NOW()
		WHERE id = $1
	`, id, strings.TrimSpace(reason))
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui status pembatalan konser: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal menyimpan pembatalan konser: %w", err)
	}

	return r.GetEventByID(ctx, id)
}

func (r *eventRepository) RescheduleEvent(ctx context.Context, id string, req *models.RescheduleEventRequest) (*models.EventItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi pengunduran jadwal konser: %w", err)
	}
	defer tx.Rollback()

	var currentStatus, currentDate, currentTime, currentOpenGate string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(status, 'SCHEDULED'), COALESCE(date, ''), COALESCE(time, ''), COALESCE(open_gate, '')
		FROM events WHERE id = $1 FOR UPDATE
	`, id).Scan(&currentStatus, &currentDate, &currentTime, &currentOpenGate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal memeriksa data konser: %w", err)
	}

	if strings.EqualFold(currentStatus, "CANCELLED") {
		return nil, ErrEventCancelled
	}

	newDate := strings.TrimSpace(req.NewDate)
	newTime := strings.TrimSpace(req.NewTime)
	newOpenGate := strings.TrimSpace(req.NewOpenGate)
	if newOpenGate == "" {
		newOpenGate = currentOpenGate
	}

	var parsedDT interface{}
	if dt, err := utils.ParseConcertDateTime(newDate, newTime); err == nil && !dt.IsZero() {
		parsedDT = dt
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE events
		SET original_date = CASE WHEN COALESCE(original_date, '') = '' THEN $2 ELSE original_date END,
		    original_time = CASE WHEN COALESCE(original_time, '') = '' THEN $3 ELSE original_time END,
		    date = $4,
		    time = $5,
		    open_gate = $6,
		    event_date_time = COALESCE($7, event_date_time),
		    reschedule_reason = $8,
		    status = 'RESCHEDULED',
		    status_updated_at = NOW()
		WHERE id = $1
	`, id, currentDate, currentTime, newDate, newTime, newOpenGate, parsedDT, strings.TrimSpace(req.Reason))
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan jadwal baru konser: %w", err)
	}

	// Sinkronkan tanggal baru pada pesanan aktif agar E-Ticket dan pencarian tiket tetap akurat
	if _, err := tx.ExecContext(ctx, `
		UPDATE orders
		SET date = $2
		WHERE event_id = $1 AND status IN ('ISSUED', 'VERIFIED', 'REMINDED')
	`, id, newDate); err != nil {
		return nil, fmt.Errorf("gagal menyinkronkan jadwal baru pada tiket pesanan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal menyimpan perubahan jadwal konser: %w", err)
	}

	return r.GetEventByID(ctx, id)
}

func insertSeatsTx(ctx context.Context, tx *sql.Tx, seats []models.SeatItem) error {
	for _, s := range seats {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO seats (id, event_id, category_id, seat_code, row_label, seat_number, status)
			VALUES ($1, $2, $3, $4, $5, $6, 'AVAILABLE')
		`, s.ID, s.EventID, s.CategoryID, s.SeatCode, s.RowLabel, s.SeatNumber); err != nil {
			return err
		}
	}
	return nil
}

func fetchExistingSeatCodesTx(ctx context.Context, tx *sql.Tx, eventID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT seat_code FROM seats WHERE event_id = $1 FOR UPDATE", eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (r *eventRepository) AddCategory(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi tambah kategori: %w", err)
	}
	defer tx.Rollback()

	var existingEventID string
	err = tx.QueryRowContext(ctx, "SELECT id FROM events WHERE id = $1", eventID).Scan(&existingEventID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal memeriksa ketersediaan konser: %w", err)
	}

	catID := fmt.Sprintf("cat-%s-%s", eventID, uuid.New().String()[:6])
	now := time.Now()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, catID, eventID, req.Name, req.Price, req.Quota, req.Quota, now)
	if err != nil {
		return nil, fmt.Errorf("gagal menambahkan kategori tiket: %w", err)
	}

	existingCodes, err := fetchExistingSeatCodesTx(ctx, tx, eventID)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca daftar kursi eksisting: %w", err)
	}

	newSeats := utils.GenerateAdditionalCategorySeats(eventID, catID, req.Quota, existingCodes)
	if err := insertSeatsTx(ctx, tx, newSeats); err != nil {
		return nil, fmt.Errorf("gagal menghasilkan nomor kursi untuk kategori baru: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal konfirmasi transaksi tambah kategori: %w", err)
	}

	return &models.TicketCategory{
		ID:             catID,
		EventID:        eventID,
		Name:           req.Name,
		Price:          req.Price,
		Quota:          req.Quota,
		RemainingQuota: req.Quota,
		CreatedAt:      now,
	}, nil
}

func (r *eventRepository) UpdateCategory(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi pembaruan kategori: %w", err)
	}
	defer tx.Rollback()

	var eventID string
	var oldQuota, oldRemaining int
	err = tx.QueryRowContext(ctx, `
		SELECT event_id, quota, remaining_quota
		FROM ticket_categories
		WHERE id = $1
		FOR UPDATE
	`, catID).Scan(&eventID, &oldQuota, &oldRemaining)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("gagal mengambil data kategori tiket: %w", err)
	}

	var bookedCount, totalSeats int
	if err := tx.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'BOOKED' THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM seats
		WHERE category_id = $1
	`, catID).Scan(&bookedCount, &totalSeats); err != nil {
		return fmt.Errorf("gagal memeriksa keterisian kursi kategori: %w", err)
	}

	if req.Quota < bookedCount {
		return ErrQuotaBelowBooked
	}

	newRemaining := oldRemaining + (req.Quota - oldQuota)
	if newRemaining < 0 {
		newRemaining = 0
	}
	if totalSeats > 0 {
		diff := req.Quota - totalSeats
		if diff > 0 {
			existingCodes, err := fetchExistingSeatCodesTx(ctx, tx, eventID)
			if err != nil {
				return fmt.Errorf("gagal membaca kursi eksisting: %w", err)
			}
			added := utils.GenerateAdditionalCategorySeats(eventID, catID, diff, existingCodes)
			if err := insertSeatsTx(ctx, tx, added); err != nil {
				return fmt.Errorf("gagal menambah kursi baru kategori: %w", err)
			}
		} else if diff < 0 {
			trimCount := -diff
			_, err := tx.ExecContext(ctx, `
				DELETE FROM seats
				WHERE id IN (
					SELECT id FROM seats
					WHERE category_id = $1 AND status = 'AVAILABLE'
					ORDER BY row_label DESC, seat_number DESC
					LIMIT $2
				)
			`, catID, trimCount)
			if err != nil {
				return fmt.Errorf("gagal memangkas kursi kategori yang belum terjual: %w", err)
			}
		}
		newRemaining = req.Quota - bookedCount
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE ticket_categories
		SET name = $1, price = $2, quota = $3, remaining_quota = $4
		WHERE id = $5
	`, req.Name, req.Price, req.Quota, newRemaining, catID)
	if err != nil {
		return fmt.Errorf("gagal memperbarui kategori tiket: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal konfirmasi pembaruan kategori tiket: %w", err)
	}

	return nil
}

func (r *eventRepository) DeleteCategory(ctx context.Context, catID string) error {
	var bookedCount int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM seats WHERE category_id = $1 AND status = 'BOOKED'", catID).Scan(&bookedCount); err != nil {
		return fmt.Errorf("gagal memeriksa status kursi kategori: %w", err)
	}
	if bookedCount > 0 {
		return ErrCategoryHasBookedSeats
	}

	result, err := r.db.ExecContext(ctx, "DELETE FROM ticket_categories WHERE id = $1", catID)
	if err != nil {
		return fmt.Errorf("gagal menghapus kategori tiket: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa baris kategori tiket yang terhapus: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *eventRepository) GetCategoriesByEventIDs(ctx context.Context, eventIDs []string) (map[string][]models.TicketCategory, error) {
	if len(eventIDs) == 0 {
		return make(map[string][]models.TicketCategory), nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_id, name, price, quota, remaining_quota, created_at
		FROM ticket_categories
		WHERE event_id = ANY($1)
		ORDER BY price DESC
	`, pq.Array(eventIDs))
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil kategori tiket secara kolektif: %w", err)
	}
	defer rows.Close()

	catMap := make(map[string][]models.TicketCategory)
	for rows.Next() {
		var cat models.TicketCategory
		if err := rows.Scan(&cat.ID, &cat.EventID, &cat.Name, &cat.Price, &cat.Quota, &cat.RemainingQuota, &cat.CreatedAt); err != nil {
			return nil, fmt.Errorf("gagal membaca data kategori tiket: %w", err)
		}
		catMap[cat.EventID] = append(catMap[cat.EventID], cat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris kategori tiket: %w", err)
	}

	return catMap, nil
}

func (r *eventRepository) GetCategoriesByEventID(ctx context.Context, eventID string) ([]models.TicketCategory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_id, name, price, quota, remaining_quota, created_at
		FROM ticket_categories
		WHERE event_id = $1
		ORDER BY price DESC
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar kategori tiket: %w", err)
	}
	defer rows.Close()

	var categories []models.TicketCategory
	for rows.Next() {
		var cat models.TicketCategory
		if err := rows.Scan(&cat.ID, &cat.EventID, &cat.Name, &cat.Price, &cat.Quota, &cat.RemainingQuota, &cat.CreatedAt); err != nil {
			return nil, fmt.Errorf("gagal membaca baris kategori tiket: %w", err)
		}
		categories = append(categories, cat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris kategori tiket: %w", err)
	}

	return categories, nil
}

func (r *eventRepository) GetSeatsByEventID(ctx context.Context, eventID string) ([]models.SeatItem, error) {
	var existingID string
	err := r.db.QueryRowContext(ctx, "SELECT id FROM events WHERE id = $1", eventID).Scan(&existingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal memeriksa data konser: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_id, category_id, seat_code, row_label, seat_number, COALESCE(order_id, ''), status
		FROM seats
		WHERE event_id = $1
		ORDER BY row_label ASC, seat_number ASC
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar kursi konser: %w", err)
	}
	defer rows.Close()

	seats := make([]models.SeatItem, 0)
	for rows.Next() {
		var s models.SeatItem
		if err := rows.Scan(&s.ID, &s.EventID, &s.CategoryID, &s.SeatCode, &s.RowLabel, &s.SeatNumber, &s.OrderID, &s.Status); err != nil {
			return nil, fmt.Errorf("gagal membaca data kursi: %w", err)
		}
		seats = append(seats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris kursi: %w", err)
	}

	return seats, nil
}
