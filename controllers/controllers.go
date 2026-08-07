
package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/services"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/lib/pq"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/v1/events (Optimized Batch Loading - Exactly 2 Queries Total)
func GetEvents(c *fiber.Ctx) error {
	rows, err := database.DB.Query(`
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
		return utils.ResponseInternalError(c, "Gagal mengambil data konser", err)
	}
	defer rows.Close()

	var events []models.EventItem
	for rows.Next() {
		var e models.EventItem
		var rundownBytes []byte
		err := rows.Scan(&e.ID, &e.Title, &e.Artist, &e.Venue, &e.Date, &e.Time, &e.Category, &e.CategoryBadgeColor, &e.Image, &e.AudioURL, &e.Conductor, &e.OpenGate, &e.Address, &e.Organizer, &e.Subtitle, &rundownBytes, &e.Description, &e.IsClosed)
		if err != nil {
			log.Printf("[GetEvents] Scan error for event %s: %v", e.ID, err)
			continue
		}
		_ = json.Unmarshal(rundownBytes, &e.Rundown)
		e.Categories = []models.TicketCategory{} // non-nil empty slice
		events = append(events, e)
	}

	// Batch loading categories in a single query filtered by current event IDs to eliminate N+1 overhead
	if len(events) > 0 {
		eventIDs := make([]string, len(events))
		for i, e := range events {
			eventIDs[i] = e.ID
		}
		catRows, err := database.DB.Query(`
			SELECT id, event_id, name, price, quota, remaining_quota, created_at
			FROM ticket_categories
			WHERE event_id = ANY($1)
			ORDER BY price DESC
		`, pq.Array(eventIDs))
		if err == nil {
			defer catRows.Close()
			catMap := make(map[string][]models.TicketCategory)
			for catRows.Next() {
				var cat models.TicketCategory
				if scanErr := catRows.Scan(&cat.ID, &cat.EventID, &cat.Name, &cat.Price, &cat.Quota, &cat.RemainingQuota, &cat.CreatedAt); scanErr == nil {
					catMap[cat.EventID] = append(catMap[cat.EventID], cat)
				}
			}
			for i := range events {
				if cats, exists := catMap[events[i].ID]; exists {
					events[i].Categories = cats
				}
			}
		}
	}

	return utils.ResponseOK(c, "Berhasil mengambil data konser", events)
}

// GET /api/v1/events/:id
func GetEventByID(c *fiber.Ctx) error {
	eventID := c.Params("id")
	var e models.EventItem
	var rundownBytes []byte
	err := database.DB.QueryRow(`
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
	`, eventID).Scan(&e.ID, &e.Title, &e.Artist, &e.Venue, &e.Date, &e.Time, &e.Category, &e.CategoryBadgeColor, &e.Image, &e.AudioURL, &e.Conductor, &e.OpenGate, &e.Address, &e.Organizer, &e.Subtitle, &rundownBytes, &e.Description, &e.IsClosed)
	_ = json.Unmarshal(rundownBytes, &e.Rundown)

	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ResponseNotFound(c, "Konser tidak ditemukan")
		}
		return utils.ResponseInternalError(c, "Gagal mengambil detail konser", err)
	}

	e.Categories = []models.TicketCategory{}
	catRows, err := database.DB.Query(`
		SELECT id, event_id, name, price, quota, remaining_quota, created_at
		FROM ticket_categories
		WHERE event_id = $1
		ORDER BY price DESC
	`, e.ID)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var cat models.TicketCategory
			if scanErr := catRows.Scan(&cat.ID, &cat.EventID, &cat.Name, &cat.Price, &cat.Quota, &cat.RemainingQuota, &cat.CreatedAt); scanErr == nil {
				e.Categories = append(e.Categories, cat)
			}
		}
	}

	return utils.ResponseOK(c, "Berhasil mengambil detail konser", e)
}

// POST /api/v1/orders (Guest Checkout dengan Row Locking & Atomic Quota Deduction)
func CreateOrder(c *fiber.Ctx) error {
	var req models.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	var userID string
	authHeader := c.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ValidateUserToken(tokenStr)
		if err == nil && claims != nil {
			userID = claims.UserID
			if req.UserEmail == "" {
				req.UserEmail = claims.Email
			}
			if req.UserName == "" {
				_ = database.DB.QueryRow("SELECT name FROM users WHERE id = $1", userID).Scan(&req.UserName)
			}
		}
	}

	if req.UserName == "" || req.UserEmail == "" || req.TicketCategoryID == "" {
		return utils.ResponseBadRequest(c, "Nama, Email, dan Kategori Tiket wajib diisi")
	}

	if req.Quantity < 1 || req.Quantity > 4 {
		return utils.ResponseBadRequest(c, "Maksimal pemesanan adalah 1 hingga 4 tiket per transaksi")
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal memulai transaksi basis data", err)
	}
	defer tx.Rollback()

	// Row locking FOR UPDATE untuk keamanan ticket war (mencegah overbooking)
	var catID, eventID, catName string
	var price float64
	var quota, remainingQuota int

	err = tx.QueryRow(`
		SELECT id, event_id, name, price, quota, remaining_quota
		FROM ticket_categories
		WHERE id = $1
		FOR UPDATE
	`, req.TicketCategoryID).Scan(&catID, &eventID, &catName, &price, &quota, &remainingQuota)

	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ResponseNotFound(c, "Kategori tiket tidak ditemukan")
		}
		return utils.ResponseInternalError(c, "Gagal memverifikasi kuota tiket", err)
	}

	if remainingQuota < req.Quantity {
		return utils.ResponseBadRequest(c, fmt.Sprintf("Kuota tiket tidak mencukupi (sisa kuota: %d)", remainingQuota))
	}

	var evtTitle, evtArtist, evtVenue, evtDate, evtTime string
	var evtClosed bool
	err = tx.QueryRow(`
		SELECT title, artist, venue, date, time, is_closed
		FROM events
		WHERE id = $1
	`, eventID).Scan(&evtTitle, &evtArtist, &evtVenue, &evtDate, &evtTime, &evtClosed)

	if err != nil {
		return utils.ResponseInternalError(c, "Gagal mengambil data event", err)
	}

	if evtClosed {
		return utils.ResponseBadRequest(c, "Penjualan tiket untuk pertunjukan ini telah ditutup karena konser sudah dimulai.")
	}

	// Potong kuota secara atomic
	newRemaining := remainingQuota - req.Quantity
	_, err = tx.Exec(`
		UPDATE ticket_categories
		SET remaining_quota = $1
		WHERE id = $2
	`, newRemaining, catID)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal memperbarui kuota tiket", err)
	}

	// Generate kode pesanan & QR Code secara kriptografis aman
	randomNum, err := utils.GenerateCryptoOTP()
	if err != nil {
		randomNum = fmt.Sprintf("%d", time.Now().UnixNano()%900000+100000)
	}
	orderCode := "SYM-" + randomNum
	orderID := uuid.New().String()
	qrCode := fmt.Sprintf("QR-%s", orderCode)
	totalPrice := price * float64(req.Quantity)
	dateFull := fmt.Sprintf("%s @ %s", evtDate, evtTime)

	// Simpan transaksi (Status awal: ISSUED)
	_, err = tx.Exec(`
		INSERT INTO orders (id, order_code, user_id, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'ISSUED', 'SANDBOX_PAYMENT')
	`, orderID, orderCode, userID, eventID, evtTitle, evtArtist, evtVenue, dateFull, catName, req.Quantity, totalPrice, req.UserName, req.UserEmail, qrCode)

	if err != nil {
		return utils.ResponseInternalError(c, "Gagal membuat pesanan tiket", err)
	}

	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal konfirmasi transaksi",
		})
	}

	resOrder := models.OrderRecord{
		ID:            orderID,
		OrderCode:     orderCode,
		UserID:        userID,
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
		Status:        "VERIFIED",
		PaymentMethod: "SANDBOX_PAYMENT",
		CreatedAt:     time.Now(),
	}

	// Kirim E-Ticket secara asinkron ke Mailpit SMTP
	services.SendETicketEmail(resOrder)

	return utils.ResponseCreated(c, "Simulasi Pembayaran Sandbox Berhasil & E-Ticket Terbit!", resOrder)
}

// GET /api/v1/tickets/lookup?code=SYM-123456 (Public Lookup Tiket Tanpa Login)
func LookupTicketByCode(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Query parameter 'code' wajib diisi",
		})
	}

	var ord models.OrderRecord
	err := database.DB.QueryRow(`
		SELECT id, order_code, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method, created_at
		FROM orders
		WHERE LOWER(order_code) = LOWER($1)
	`, code).Scan(&ord.ID, &ord.OrderCode, &ord.EventID, &ord.EventTitle, &ord.Artist, &ord.Venue, &ord.Date, &ord.CategoryName, &ord.Quantity, &ord.TotalPrice, &ord.UserName, &ord.UserEmail, &ord.QRCode, &ord.Status, &ord.PaymentMethod, &ord.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kode pesanan tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal melakukan pencarian tiket",
			Error:   err.Error(),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Tiket ditemukan",
		Data:    ord,
	})
}

// POST /api/v1/admin/login
func AdminLogin(c *fiber.Ctx) error {
	var req models.AdminLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request login tidak valid",
			Error:   err.Error(),
		})
	}

	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}

	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "123"
	}

	if req.Username != adminUser || req.Password != adminPass {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Username atau Password Admin salah",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Login Admin berhasil",
		Data: fiber.Map{
			"username": adminUser,
			"token":    "admin-session-token-symphoniatic-2026",
		},
	})
}

// POST /api/v1/admin/events (Create Event + Ticket Categories)
func CreateEvent(c *fiber.Ctx) error {
	var req models.CreateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if req.Title == "" || req.Artist == "" || req.Venue == "" || req.Date == "" || req.Time == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Judul, Artist, Venue, Tanggal, dan Waktu wajib diisi",
		})
	}

	if req.Category == "" {
		req.Category = "SIMFONI UTAMA"
	}
	if req.CategoryBadgeColor == "" {
		req.CategoryBadgeColor = "bg-blue-900/80 text-blue-200 border-blue-500/40"
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memulai transaksi",
		})
	}
	defer tx.Rollback()

	eventID := fmt.Sprintf("evt-%s", uuid.New().String()[:8])
	rundownJSON, _ := json.Marshal(req.Rundown)
	if req.Rundown == nil {
		rundownJSON = []byte("[]")
	}
	_, err = tx.Exec(`
		INSERT INTO events (id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, conductor, open_gate, address, organizer, subtitle, rundown, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, eventID, req.Title, req.Artist, req.Venue, req.Date, req.Time, req.Category, req.CategoryBadgeColor, req.Image, req.AudioURL, req.Conductor, req.OpenGate, req.Address, req.Organizer, req.Subtitle, string(rundownJSON), req.Description)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menyimpan event konser ke database",
			Error:   err.Error(),
		})
	}

	// Insert categories if provided
	var createdCats []models.TicketCategory
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

		_, err = tx.Exec(`
			INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, catID, eventID, catName, catPrice, catQuota, catQuota)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Message: "Gagal menyimpan kategori tiket",
				Error:   err.Error(),
			})
		}

		createdCats = append(createdCats, models.TicketCategory{
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
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal konfirmasi transaksi event",
		})
	}

	resEvent := models.EventItem{
		ID:                 eventID,
		Title:              req.Title,
		Artist:             req.Artist,
		Venue:              req.Venue,
		Date:               req.Date,
		Time:               req.Time,
		Category:           req.Category,
		CategoryBadgeColor: req.CategoryBadgeColor,
		Image:              req.Image,
		AudioURL:           req.AudioURL,
		Conductor:          req.Conductor,
		OpenGate:           req.OpenGate,
		Address:            req.Address,
		Organizer:          req.Organizer,
		Subtitle:           req.Subtitle,
		Description:        req.Description,
		Rundown:            req.Rundown,
		Categories:         createdCats,
	}

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Event konser berhasil ditambahkan",
		Data:    resEvent,
	})
}

// PUT /api/v1/admin/events/:id (Update Event)
func UpdateEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")
	var req models.UpdateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	rundownJSON, _ := json.Marshal(req.Rundown)
	if req.Rundown == nil {
		rundownJSON = []byte("[]")
	}
	res, err := database.DB.Exec(`
		UPDATE events
		SET title = $1, artist = $2, venue = $3, date = $4, time = $5, category = $6, category_badge_color = $7, image = $8, audio_url = $9, conductor = $10, open_gate = $11, address = $12, organizer = $13, subtitle = $14, rundown = $15, description = $16
		WHERE id = $17
	`, req.Title, req.Artist, req.Venue, req.Date, req.Time, req.Category, req.CategoryBadgeColor, req.Image, req.AudioURL, req.Conductor, req.OpenGate, req.Address, req.Organizer, req.Subtitle, string(rundownJSON), req.Description, eventID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui event konser",
			Error:   err.Error(),
		})
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Event konser tidak ditemukan",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Event konser berhasil diperbarui",
	})
}

// DELETE /api/v1/admin/events/:id (Delete Event)
func DeleteEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")
	res, err := database.DB.Exec("DELETE FROM events WHERE id = $1", eventID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menghapus event konser",
			Error:   err.Error(),
		})
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Event konser tidak ditemukan",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Event konser dan kategori tiket terkait berhasil dihapus",
	})
}

// POST /api/v1/admin/events/:id/categories (Add Ticket Category)
func CreateTicketCategory(c *fiber.Ctx) error {
	eventID := c.Params("id")
	var req models.CreateCategoryInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if req.Name == "" || req.Price <= 0 || req.Quota <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Nama, Harga, dan Kuota tiket wajib valid",
		})
	}

	catID := fmt.Sprintf("cat-%s-%s", eventID, uuid.New().String()[:6])
	_, err := database.DB.Exec(`
		INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, catID, eventID, req.Name, req.Price, req.Quota, req.Quota)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menambah kategori tiket",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil ditambahkan",
		Data: models.TicketCategory{
			ID:             catID,
			EventID:        eventID,
			Name:           req.Name,
			Price:          req.Price,
			Quota:          req.Quota,
			RemainingQuota: req.Quota,
			CreatedAt:      time.Now(),
		},
	})
}

// PUT /api/v1/admin/categories/:id (Update Ticket Category)
func UpdateTicketCategory(c *fiber.Ctx) error {
	catID := c.Params("id")
	var req models.UpdateTicketCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	res, err := database.DB.Exec(`
		UPDATE ticket_categories
		SET name = $1, price = $2, remaining_quota = GREATEST(0, remaining_quota + ($3 - quota)), quota = $3
		WHERE id = $4
	`, req.Name, req.Price, req.Quota, catID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui kategori tiket",
			Error:   err.Error(),
		})
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Kategori tiket tidak ditemukan",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil diperbarui",
	})
}

// DELETE /api/v1/admin/categories/:id (Delete Category)
func DeleteTicketCategory(c *fiber.Ctx) error {
	catID := c.Params("id")
	res, err := database.DB.Exec("DELETE FROM ticket_categories WHERE id = $1", catID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menghapus kategori tiket",
			Error:   err.Error(),
		})
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Kategori tiket tidak ditemukan",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil dihapus",
	})
}

// GET /api/v1/admin/orders (List all orders with optional filter & search)
func GetAllOrders(c *fiber.Ctx) error {
	search := c.Query("search")
	status := c.Query("status")

	query := `
		SELECT id, order_code, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method, created_at
		FROM orders
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if search != "" {
		query += fmt.Sprintf(" AND (LOWER(order_code) LIKE $%d OR LOWER(user_name) LIKE $%d OR LOWER(user_email) LIKE $%d OR LOWER(event_title) LIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, strings.ToUpper(status))
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil daftar pesanan",
			Error:   err.Error(),
		})
	}
	defer rows.Close()

	var ordersList []models.OrderRecord
	for rows.Next() {
		var ord models.OrderRecord
		err := rows.Scan(&ord.ID, &ord.OrderCode, &ord.EventID, &ord.EventTitle, &ord.Artist, &ord.Venue, &ord.Date, &ord.CategoryName, &ord.Quantity, &ord.TotalPrice, &ord.UserName, &ord.UserEmail, &ord.QRCode, &ord.Status, &ord.PaymentMethod, &ord.CreatedAt)
		if err == nil {
			ordersList = append(ordersList, ord)
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Daftar pesanan berhasil diambil",
		Data:    ordersList,
	})
}

// PATCH /api/v1/admin/orders/:id/status (Update Order Status)
func UpdateOrderStatus(c *fiber.Ctx) error {
	orderID := c.Params("id")
	var req models.UpdateOrderStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Status pesanan wajib diisi",
		})
	}

	res, err := database.DB.Exec("UPDATE orders SET status = $1 WHERE id = $2", strings.ToUpper(req.Status), orderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status pesanan",
			Error:   err.Error(),
		})
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Pesanan tidak ditemukan",
		})
	}

	if strings.ToUpper(req.Status) == "REMINDED" || strings.ToUpper(req.Status) == "REMINDER" {
		var userEmail, userName, eventTitle, venue, date, orderCode string
		err := database.DB.QueryRow("SELECT user_email, user_name, event_title, venue, date, order_code FROM orders WHERE id = $1", orderID).Scan(&userEmail, &userName, &eventTitle, &venue, &date, &orderCode)
		if err == nil {
			services.SendEventReminderEmail(userEmail, userName, eventTitle, venue, "", date, orderCode)
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Status pesanan berhasil diperbarui menjadi %s", req.Status),
	})
}

// GET /api/v1/admin/dashboard (Enhanced Admin Dashboard Analytics)
func GetAdminDashboardMetrics(c *fiber.Ctx) error {
	var totalRevenue float64
	var ticketsSold int
	var remainingQuota int
	var totalEvents int
	var totalOrders int

	_ = database.DB.QueryRow(`
		SELECT
			COALESCE((SELECT SUM(total_price) FROM orders WHERE status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')), 0),
			COALESCE((SELECT SUM(quantity) FROM orders WHERE status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')), 0),
			COALESCE((SELECT COUNT(*) FROM orders WHERE status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')), 0),
			COALESCE((SELECT SUM(remaining_quota) FROM ticket_categories), 0),
			COALESCE((SELECT COUNT(*) FROM events), 0)
	`).Scan(&totalRevenue, &ticketsSold, &totalOrders, &remainingQuota, &totalEvents)

	// Revenue by Event breakdown
	rows, err := database.DB.Query(`
		SELECT e.id, e.title, COALESCE(SUM(o.total_price), 0) as rev, COALESCE(SUM(o.quantity), 0) as sold
		FROM events e
		LEFT JOIN orders o ON e.id = o.event_id AND o.status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')
		GROUP BY e.id, e.title
		ORDER BY rev DESC
	`)

	type EventRevenueStat struct {
		EventID     string  `json:"eventId"`
		Title       string  `json:"title"`
		Revenue     float64 `json:"revenue"`
		TicketsSold int     `json:"ticketsSold"`
	}

	var eventStats []EventRevenueStat
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s EventRevenueStat
			if scanErr := rows.Scan(&s.EventID, &s.Title, &s.Revenue, &s.TicketsSold); scanErr == nil {
				eventStats = append(eventStats, s)
			}
		}
	}

	// Query Revenue Timeline (Last 6 Months)
	timelineRows, err := database.DB.Query(`
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

	type TimelineStat struct {
		Month   string  `json:"month"`
		Revenue float64 `json:"revenue"`
		Tickets int     `json:"tickets"`
	}

	// Pre-populate last 6 months back from today as default
	revenueTimeline := make([]TimelineStat, 6)
	now := time.Now()
	for i := 5; i >= 0; i-- {
		m := now.AddDate(0, -i, 0)
		revenueTimeline[5-i] = TimelineStat{
			Month:   m.Format("Jan"),
			Revenue: 0,
			Tickets: 0,
		}
	}

	if err == nil {
		defer timelineRows.Close()
		for timelineRows.Next() {
			var t TimelineStat
			if scanErr := timelineRows.Scan(&t.Month, &t.Revenue, &t.Tickets); scanErr == nil {
				// Overwrite matching month
				for idx, item := range revenueTimeline {
					if strings.EqualFold(item.Month, t.Month) {
						revenueTimeline[idx].Revenue = t.Revenue
						revenueTimeline[idx].Tickets = t.Tickets
						break
					}
				}
			}
		}
	}

	// Query Category Distribution
	catRows, err := database.DB.Query(`
		SELECT 
			category_name, 
			COALESCE(SUM(total_price), 0) as total_val
		FROM orders
		WHERE status IN ('ISSUED', 'VERIFIED', 'CHECKED_IN')
		GROUP BY category_name
		ORDER BY total_val DESC
	`)

	type CategoryStat struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
	}

	var categoryDistribution []CategoryStat
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var c CategoryStat
			if scanErr := catRows.Scan(&c.Name, &c.Value); scanErr == nil {
				categoryDistribution = append(categoryDistribution, c)
			}
		}
	}

	// Recent 5 Orders
	recentRows, err := database.DB.Query(`
		SELECT id, order_code, event_title, quantity, total_price, user_name, status, created_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT 5
	`)

	type RecentOrderSummary struct {
		ID         string    `json:"id"`
		OrderCode  string    `json:"orderCode"`
		EventTitle string    `json:"eventTitle"`
		Quantity   int       `json:"quantity"`
		TotalPrice float64   `json:"totalPrice"`
		UserName   string    `json:"userName"`
		Status     string    `json:"status"`
		CreatedAt  time.Time `json:"createdAt"`
	}

	var recentOrders []RecentOrderSummary
	if err == nil {
		defer recentRows.Close()
		for recentRows.Next() {
			var ro RecentOrderSummary
			if scanErr := recentRows.Scan(&ro.ID, &ro.OrderCode, &ro.EventTitle, &ro.Quantity, &ro.TotalPrice, &ro.UserName, &ro.Status, &ro.CreatedAt); scanErr == nil {
				recentOrders = append(recentOrders, ro)
			}
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Metrik admin berhasil diambil",
		Data: fiber.Map{
			"totalRevenue":         totalRevenue,
			"ticketsSold":          ticketsSold,
			"remainingQuota":       remainingQuota,
			"totalEvents":          totalEvents,
			"totalOrders":          totalOrders,
			"eventStats":           eventStats,
			"recentOrders":         recentOrders,
			"revenueTimeline":      revenueTimeline,
			"categoryDistribution": categoryDistribution,
		},
	})
}

// ─── REFUND CONTROLLERS ───

// POST /api/v1/refunds/request-otp
func RequestRefundOTP(c *fiber.Ctx) error {
	var input models.RequestRefundOTPInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
			Error:   err.Error(),
		})
	}

	orderCode := strings.TrimSpace(input.OrderCode)
	userEmail := strings.ToLower(strings.TrimSpace(input.UserEmail))

	if orderCode == "" || userEmail == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode pesanan dan email wajib diisi",
		})
	}

	// 1. Check order existence and ownership
	var order models.OrderRecord
	err := database.DB.QueryRow(`
		SELECT id, order_code, event_title, user_email, status, total_price, quantity
		FROM orders
		WHERE UPPER(order_code) = UPPER($1)
	`, orderCode).Scan(&order.ID, &order.OrderCode, &order.EventTitle, &order.UserEmail, &order.Status, &order.TotalPrice, &order.Quantity)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kode pesanan tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memverifikasi pesanan",
			Error:   err.Error(),
		})
	}

	// Email check
	if !strings.EqualFold(order.UserEmail, userEmail) {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Email tidak cocok dengan pemegang tiket ini",
		})
	}

	// Check order status
	if order.Status == "CHECKED_IN" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tiket sudah di-scan (Check In) pada venue dan tidak dapat dikembalikan / refund",
		})
	}
	if order.Status == "REFUND_REQUESTED" || order.Status == "REFUNDED" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Pengajuan refund untuk tiket ini sedang diproses atau sudah selesai",
		})
	}
	if order.Status == "CANCELLED" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Pesanan ini sudah dibatalkan sebelumnya",
		})
	}

	// Generate 6-digit OTP
	otpCode := fmt.Sprintf("%06d", rand.Intn(1000000))
	expiresAt := time.Now().Add(10 * time.Minute)

	// Check if existing refund_request record exists
	var existingID string
	err = database.DB.QueryRow(`
		SELECT id FROM refund_requests WHERE order_id = $1
	`, order.ID).Scan(&existingID)

	if err == sql.ErrNoRows {
		refundID := "rf-" + uuid.New().String()
		_, err = database.DB.Exec(`
			INSERT INTO refund_requests (id, order_id, order_code, user_email, bank_name, account_number, account_holder, reason, refund_amount, status, otp_code, otp_expires_at)
			VALUES ($1, $2, $3, $4, '', '', '', '', $5, 'OTP_SENT', $6, $7)
		`, refundID, order.ID, order.OrderCode, order.UserEmail, order.TotalPrice, otpCode, expiresAt)
	} else if err == nil {
		_, err = database.DB.Exec(`
			UPDATE refund_requests
			SET otp_code = $1, otp_expires_at = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`, otpCode, expiresAt, existingID)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat kode verifikasi OTP",
			Error:   err.Error(),
		})
	}

	// Send OTP email
	services.SendRefundOTPEmail(order.UserEmail, order.OrderCode, otpCode)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Kode OTP verifikasi refund telah dikirim ke email " + order.UserEmail,
		Data: fiber.Map{
			"orderCode": order.OrderCode,
			"userEmail": order.UserEmail,
			"expiresIn": "10 Menit",
		},
	})
}

// POST /api/v1/refunds/submit
func SubmitRefund(c *fiber.Ctx) error {
	var input models.SubmitRefundInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
			Error:   err.Error(),
		})
	}

	orderCode := strings.TrimSpace(input.OrderCode)
	userEmail := strings.ToLower(strings.TrimSpace(input.UserEmail))
	otpCode := strings.TrimSpace(input.OTPCode)
	bankName := strings.TrimSpace(input.BankName)
	accountNumber := strings.TrimSpace(input.AccountNumber)
	accountHolder := strings.TrimSpace(input.AccountHolder)
	reason := strings.TrimSpace(input.Reason)

	if orderCode == "" || userEmail == "" || otpCode == "" || bankName == "" || accountNumber == "" || accountHolder == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Seluruh kolom formulir dan OTP wajib diisi",
		})
	}

	// Query refund_request record
	var rr models.RefundRequestRecord
	var dbOTP string
	var expiresAt time.Time
	var currentStatus string

	err := database.DB.QueryRow(`
		SELECT id, order_id, order_code, user_email, status, otp_code, otp_expires_at
		FROM refund_requests
		WHERE UPPER(order_code) = UPPER($1)
	`, orderCode).Scan(&rr.ID, &rr.OrderID, &rr.OrderCode, &rr.UserEmail, &currentStatus, &dbOTP, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Pengajuan refund tidak ditemukan. Silakan minta kode OTP terlebih dahulu.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memproses data refund",
			Error:   err.Error(),
		})
	}

	if !strings.EqualFold(rr.UserEmail, userEmail) {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Email tidak sesuai dengan pemegang tiket ini",
		})
	}

	if dbOTP != otpCode {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode OTP yang Anda masukkan salah",
		})
	}

	if time.Now().After(expiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode OTP telah kadaluarsa. Silakan minta kode OTP baru.",
		})
	}

	// Start SQL Transaction
	tx, err := database.DB.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memulai transaksi database",
			Error:   err.Error(),
		})
	}
	defer tx.Rollback()

	// Update refund_requests to PENDING
	_, err = tx.Exec(`
		UPDATE refund_requests
		SET bank_name = $1, account_number = $2, account_holder = $3, reason = $4, status = 'PENDING', otp_code = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`, bankName, accountNumber, accountHolder, reason, rr.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengupdate permohonan refund",
			Error:   err.Error(),
		})
	}

	// Update orders status to REFUND_REQUESTED
	_, err = tx.Exec(`
		UPDATE orders
		SET status = 'REFUND_REQUESTED'
		WHERE id = $1
	`, rr.OrderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status pesanan",
			Error:   err.Error(),
		})
	}

	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menyimpan pengajuan refund",
			Error:   err.Error(),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Pengajuan refund berhasil dikirim. Tim Finance akan meninjau dan memproses pengembalian dana Anda.",
		Data: fiber.Map{
			"orderCode":     rr.OrderCode,
			"status":        "PENDING",
			"bankName":      bankName,
			"accountHolder": accountHolder,
		},
	})
}

// POST /api/v1/refunds/status
func GetRefundStatus(c *fiber.Ctx) error {
	var input models.CheckRefundStatusRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
		})
	}

	orderCode := strings.TrimSpace(input.OrderCode)
	userEmail := strings.ToLower(strings.TrimSpace(input.UserEmail))

	if orderCode == "" || userEmail == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode pesanan dan email wajib diisi",
		})
	}

	var rr models.RefundRequestRecord
	var orderStatus string

	err := database.DB.QueryRow(`
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, r.account_holder, r.reason, r.refund_amount, r.status, COALESCE(r.admin_note, ''), r.created_at, r.updated_at, o.status, o.event_title, o.category_name, o.quantity, o.user_name
		FROM refund_requests r
		JOIN orders o ON o.id = r.order_id
		WHERE UPPER(r.order_code) = UPPER($1) AND LOWER(r.user_email) = LOWER($2)
	`, orderCode, userEmail).Scan(
		&rr.ID, &rr.OrderID, &rr.OrderCode, &rr.UserEmail, &rr.BankName, &rr.AccountNumber, &rr.AccountHolder,
		&rr.Reason, &rr.RefundAmount, &rr.Status, &rr.AdminNote, &rr.CreatedAt, &rr.UpdatedAt,
		&orderStatus, &rr.EventTitle, &rr.CategoryName, &rr.Quantity, &rr.UserName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Data pengajuan refund tidak ditemukan untuk kode dan email ini",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil status refund",
			Error:   err.Error(),
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil mengambil status refund",
		Data: fiber.Map{
			"orderStatus":  orderStatus,
			"refundDetail": rr,
		},
	})
}

// GET /api/v1/admin/refunds
func AdminGetAllRefunds(c *fiber.Ctx) error {
	rows, err := database.DB.Query(`
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, r.account_holder, r.reason, r.refund_amount, r.status, COALESCE(r.admin_note, ''), r.created_at, r.updated_at, o.event_title, o.category_name, o.quantity, o.user_name
		FROM refund_requests r
		JOIN orders o ON o.id = r.order_id
		WHERE r.status != 'OTP_SENT'
		ORDER BY r.created_at DESC
	`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil daftar permohonan refund",
			Error:   err.Error(),
		})
	}
	defer rows.Close()

	var refunds []models.RefundRequestRecord
	for rows.Next() {
		var rr models.RefundRequestRecord
		err := rows.Scan(
			&rr.ID, &rr.OrderID, &rr.OrderCode, &rr.UserEmail, &rr.BankName, &rr.AccountNumber, &rr.AccountHolder,
			&rr.Reason, &rr.RefundAmount, &rr.Status, &rr.AdminNote, &rr.CreatedAt, &rr.UpdatedAt,
			&rr.EventTitle, &rr.CategoryName, &rr.Quantity, &rr.UserName,
		)
		if err == nil {
			refunds = append(refunds, rr)
		}
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil mengambil daftar permohonan refund",
		Data:    refunds,
	})
}

// PATCH /api/v1/admin/refunds/:id/status
func AdminUpdateRefundStatus(c *fiber.Ctx) error {
	refundID := c.Params("id")
	var input models.UpdateRefundStatusRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
		})
	}

	newStatus := strings.ToUpper(strings.TrimSpace(input.Status))
	if newStatus != "APPROVED" && newStatus != "REJECTED" && newStatus != "COMPLETED" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Status harus APPROVED, REJECTED, atau COMPLETED",
		})
	}

	// 1. Fetch existing refund details
	var orderID, orderCode, userEmail, categoryName, eventID string
	var quantity int
	var amount float64

	err := database.DB.QueryRow(`
		SELECT r.order_id, r.order_code, r.user_email, r.refund_amount, o.event_id, o.category_name, o.quantity
		FROM refund_requests r
		JOIN orders o ON o.id = r.order_id
		WHERE r.id = $1
	`, refundID).Scan(&orderID, &orderCode, &userEmail, &amount, &eventID, &categoryName, &quantity)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Data refund tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil detail refund",
			Error:   err.Error(),
		})
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat transaksi DB",
		})
	}
	defer tx.Rollback()

	// Update refund record
	_, err = tx.Exec(`
		UPDATE refund_requests
		SET status = $1, admin_note = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, newStatus, input.AdminNote, refundID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status refund",
			Error:   err.Error(),
		})
	}

	if newStatus == "APPROVED" || newStatus == "COMPLETED" {
		// Update order status to REFUNDED
		_, err = tx.Exec(`
			UPDATE orders SET status = 'REFUNDED' WHERE id = $1
		`, orderID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Message: "Gagal memperbarui status order",
				Error:   err.Error(),
			})
		}

		// RESTOCK QUOTA
		_, err = tx.Exec(`
			UPDATE ticket_categories
			SET remaining_quota = remaining_quota + $1
			WHERE event_id = $2 AND name = $3
		`, quantity, eventID, categoryName)
		if err != nil {
			log.Printf("[WARNING] Restock kuota tiket gagal: %v", err)
		}
	} else if newStatus == "REJECTED" {
		// Restore order status to VERIFIED
		_, err = tx.Exec(`
			UPDATE orders SET status = 'VERIFIED' WHERE id = $1
		`, orderID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Message: "Gagal mengembalikan status order",
				Error:   err.Error(),
			})
		}
	}

	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal melakukan commit transaksi",
			Error:   err.Error(),
		})
	}

	// Send status email notification
	services.SendRefundStatusNotificationEmail(userEmail, orderCode, newStatus, input.AdminNote, amount)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Status pengajuan refund berhasil diubah menjadi %s", newStatus),
	})
}

// PATCH /api/v1/admin/events/:id/toggle-close
func ToggleEventClose(c *fiber.Ctx) error {
	eventID := c.Params("id")
	var isClosed bool
	err := database.DB.QueryRow(`SELECT is_closed FROM events WHERE id = $1`, eventID).Scan(&isClosed)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Konser tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengecek status konser",
			Error:   err.Error(),
		})
	}

	newClosedState := !isClosed
	_, err = database.DB.Exec(`UPDATE events SET is_closed = $1 WHERE id = $2`, newClosedState, eventID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status penutupan konser",
			Error:   err.Error(),
		})
	}

	msg := "Penjualan tiket konser berhasil ditutup"
	if !newClosedState {
		msg = "Penjualan tiket konser berhasil dibuka kembali"
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: msg,
		Data: fiber.Map{
			"eventId":  eventID,
			"isClosed": newClosedState,
		},
	})
}

// POST /api/v1/admin/upload
func UploadImage(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membaca file gambar",
			Error:   err.Error(),
		})
	}

	// Create uploads directory if not exists
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat folder penyimpanan",
			Error:   err.Error(),
		})
	}

	// Unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filepath := fmt.Sprintf("./uploads/%s", filename)

	if err := c.SaveFile(file, filepath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menyimpan file gambar",
			Error:   err.Error(),
		})
	}

	// Dynamic base URL or fallback to localhost
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8082"
	} else {
		// remove /api/v1 prefix from base URL if it's there
		baseURL = strings.TrimSuffix(baseURL, "/api/v1")
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "File gambar berhasil diunggah",
		Data: fiber.Map{
			"url": fmt.Sprintf("%s/uploads/%s", baseURL, filename),
		},
	})
}


