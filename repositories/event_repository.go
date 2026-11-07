package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
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
			COALESCE(is_closed, FALSE)
		FROM events
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data konser: %w", err)
	}
	defer rows.Close()

	var eventList []models.EventItem
	for rows.Next() {
		var eventItem models.EventItem
		var rundownBytes []byte

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
		); err != nil {
			return nil, fmt.Errorf("gagal membaca baris data konser: %w", err)
		}
		_ = json.Unmarshal(rundownBytes, &eventItem.Rundown)
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
			COALESCE(is_closed, FALSE)
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
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal mengambil detail konser: %w", err)
	}

	_ = json.Unmarshal(rundownBytes, &eventItem.Rundown)

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

	_, err = tx.ExecContext(ctx, `
		INSERT INTO events (id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, conductor, open_gate, address, organizer, subtitle, rundown, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, eventID, req.Title, req.Artist, req.Venue, req.Date, req.Time, req.Category, badgeColor, req.Image, req.AudioURL, req.Conductor, req.OpenGate, req.Address, req.Organizer, req.Subtitle, string(rundownJSON), req.Description)

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

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal konfirmasi transaksi pembuatan konser: %w", err)
	}

	return &models.EventItem{
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
		Rundown:            req.Rundown,
		Categories:         createdCategories,
	}, nil
}

func (r *eventRepository) UpdateEvent(ctx context.Context, id string, req *models.UpdateEventRequest) error {
	rundownJSON, _ := json.Marshal(req.Rundown)
	if req.Rundown == nil {
		rundownJSON = []byte("[]")
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE events
		SET title = $1, artist = $2, venue = $3, date = $4, time = $5, category = $6, 
		    category_badge_color = $7, image = $8, audio_url = $9, conductor = $10, 
		    open_gate = $11, address = $12, organizer = $13, subtitle = $14, rundown = $15, description = $16
		WHERE id = $17
	`, req.Title, req.Artist, req.Venue, req.Date, req.Time, req.Category, req.CategoryBadgeColor, req.Image, req.AudioURL, req.Conductor, req.OpenGate, req.Address, req.Organizer, req.Subtitle, string(rundownJSON), req.Description, id)

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
	err := r.db.QueryRowContext(ctx, "SELECT is_closed FROM events WHERE id = $1", id).Scan(&isClosed)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrNotFound
		}
		return false, fmt.Errorf("gagal memeriksa status penutupan konser: %w", err)
	}

	newState := !isClosed
	_, err = r.db.ExecContext(ctx, "UPDATE events SET is_closed = $1 WHERE id = $2", newState, id)
	if err != nil {
		return false, fmt.Errorf("gagal mengubah status penutupan konser: %w", err)
	}

	return newState, nil
}

func (r *eventRepository) AddCategory(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error) {
	var existingEventID string
	err := r.db.QueryRowContext(ctx, "SELECT id FROM events WHERE id = $1", eventID).Scan(&existingEventID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("gagal memeriksa ketersediaan konser: %w", err)
	}

	catID := fmt.Sprintf("cat-%s-%s", eventID, uuid.New().String()[:6])
	now := time.Now()

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, catID, eventID, req.Name, req.Price, req.Quota, req.Quota, now)

	if err != nil {
		return nil, fmt.Errorf("gagal menambahkan kategori tiket: %w", err)
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
	// Formula GREATEST(0, remaining_quota + (req.Quota - quota)) menjaga sinkronisasi selisih kuota baru tanpa membiarkan kuota bernilai negatif jika tiket telah terjual melebihi kuota revisi
	result, err := r.db.ExecContext(ctx, `
		UPDATE ticket_categories
		SET name = $1, price = $2, remaining_quota = GREATEST(0, remaining_quota + ($3 - quota)), quota = $3
		WHERE id = $4
	`, req.Name, req.Price, req.Quota, catID)

	if err != nil {
		return fmt.Errorf("gagal memperbarui kategori tiket: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa baris kategori tiket yang diperbarui: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *eventRepository) DeleteCategory(ctx context.Context, catID string) error {
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
