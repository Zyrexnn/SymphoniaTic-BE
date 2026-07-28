package controllers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/v1/events
func GetEvents(c *fiber.Ctx) error {
	rows, err := database.DB.Query(`
		SELECT id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, description
		FROM events
		ORDER BY created_at ASC
	`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil data konser",
			Error:   err.Error(),
		})
	}
	defer rows.Close()

	var events []models.EventItem
	for rows.Next() {
		var e models.EventItem
		err := rows.Scan(&e.ID, &e.Title, &e.Artist, &e.Venue, &e.Date, &e.Time, &e.Category, &e.CategoryBadgeColor, &e.Image, &e.AudioURL, &e.Description)
		if err != nil {
			continue
		}

		catRows, err := database.DB.Query(`
			SELECT id, event_id, name, price, quota, remaining_quota, created_at
			FROM ticket_categories
			WHERE event_id = $1
			ORDER BY price DESC
		`, e.ID)
		if err == nil {
			var cats []models.TicketCategory
			for catRows.Next() {
				var cat models.TicketCategory
				if scanErr := catRows.Scan(&cat.ID, &cat.EventID, &cat.Name, &cat.Price, &cat.Quota, &cat.RemainingQuota, &cat.CreatedAt); scanErr == nil {
					cats = append(cats, cat)
				}
			}
			catRows.Close()
			e.Categories = cats
		}

		events = append(events, e)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil mengambil data konser",
		Data:    events,
	})
}

// GET /api/v1/events/:id
func GetEventByID(c *fiber.Ctx) error {
	eventID := c.Params("id")
	var e models.EventItem
	err := database.DB.QueryRow(`
		SELECT id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, description
		FROM events
		WHERE id = $1
	`, eventID).Scan(&e.ID, &e.Title, &e.Artist, &e.Venue, &e.Date, &e.Time, &e.Category, &e.CategoryBadgeColor, &e.Image, &e.AudioURL, &e.Description)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Konser tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil detail konser",
			Error:   err.Error(),
		})
	}

	catRows, err := database.DB.Query(`
		SELECT id, event_id, name, price, quota, remaining_quota, created_at
		FROM ticket_categories
		WHERE event_id = $1
		ORDER BY price DESC
	`, e.ID)
	if err == nil {
		var cats []models.TicketCategory
		for catRows.Next() {
			var cat models.TicketCategory
			if scanErr := catRows.Scan(&cat.ID, &cat.EventID, &cat.Name, &cat.Price, &cat.Quota, &cat.RemainingQuota, &cat.CreatedAt); scanErr == nil {
				cats = append(cats, cat)
			}
		}
		catRows.Close()
		e.Categories = cats
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil mengambil detail konser",
		Data:    e,
	})
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

	if req.UserName == "" || req.UserEmail == "" || req.TicketCategoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Nama, Email, dan Kategori Tiket wajib diisi",
		})
	}

	if req.Quantity < 1 || req.Quantity > 4 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Maksimal pemesanan adalah 1 hingga 4 tiket per transaksi",
		})
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memulai transaksi basis data",
		})
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
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kategori tiket tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memverifikasi kuota tiket",
		})
	}

	if remainingQuota < req.Quantity {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: fmt.Sprintf("Kuota tiket tidak mencukupi (sisa kuota: %d)", remainingQuota),
		})
	}

	var evtTitle, evtArtist, evtVenue, evtDate, evtTime string
	err = tx.QueryRow(`
		SELECT title, artist, venue, date, time
		FROM events
		WHERE id = $1
	`, eventID).Scan(&evtTitle, &evtArtist, &evtVenue, &evtDate, &evtTime)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil data event",
		})
	}

	// Potong kuota secara atomic
	newRemaining := remainingQuota - req.Quantity
	_, err = tx.Exec(`
		UPDATE ticket_categories
		SET remaining_quota = $1
		WHERE id = $2
	`, newRemaining, catID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui kuota tiket",
		})
	}

	// Generate kode pesanan & QR Code
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	orderCode := fmt.Sprintf("SYM-%d", r.Intn(900000)+100000)
	orderID := uuid.New().String()
	qrCode := fmt.Sprintf("QR-%s", orderCode)
	totalPrice := price * float64(req.Quantity)
	dateFull := fmt.Sprintf("%s @ %s", evtDate, evtTime)

	// Simpan transaksi (Sandbox Auto-Verified Payment Simulation)
	_, err = tx.Exec(`
		INSERT INTO orders (id, order_code, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'VERIFIED', 'SANDBOX_PAYMENT')
	`, orderID, orderCode, eventID, evtTitle, evtArtist, evtVenue, dateFull, catName, req.Quantity, totalPrice, req.UserName, req.UserEmail, qrCode)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat pesanan tiket",
			Error:   err.Error(),
		})
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

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Simulasi Pembayaran Sandbox Berhasil & E-Ticket Terbit!",
		Data:    resOrder,
	})
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

// GET /api/v1/admin/dashboard
func GetAdminDashboardMetrics(c *fiber.Ctx) error {
	var totalRevenue float64
	var ticketsSold int
	var remainingQuota int

	_ = database.DB.QueryRow("SELECT COALESCE(SUM(total_price), 0), COALESCE(SUM(quantity), 0) FROM orders WHERE status = 'VERIFIED'").Scan(&totalRevenue, &ticketsSold)
	_ = database.DB.QueryRow("SELECT COALESCE(SUM(remaining_quota), 0) FROM ticket_categories").Scan(&remainingQuota)

	return c.JSON(models.APIResponse{
		Success: true,
		Message: "Metrik admin berhasil diambil",
		Data: fiber.Map{
			"totalRevenue":   totalRevenue,
			"ticketsSold":    ticketsSold,
			"remainingQuota": remainingQuota,
		},
	})
}
