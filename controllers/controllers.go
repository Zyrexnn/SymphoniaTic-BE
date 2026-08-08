
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
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// GetEvents godoc
// @Summary Mengambil daftar seluruh konser
// @Description Mengambil daftar seluruh event/konser yang tersedia lengkap dengan kategori tiket dan rundown.
// @Tags Public - Events
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=[]models.EventItem} "Berhasil mengambil data konser"
// @Failure 500 {object} models.APIResponse "Gagal mengambil data konser"
// @Router /events [get]
func GetEvents(ctx *fiber.Ctx) error {
	eventRows, err := database.DB.Query(`
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
		return utils.ResponseInternalError(ctx, "Gagal mengambil data konser", err)
	}
	defer eventRows.Close()

	var eventList []models.EventItem
	for eventRows.Next() {
		var eventItem models.EventItem
		var rundownBytes []byte

		if scanErr := eventRows.Scan(
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
		); scanErr != nil {
			log.Println("Scan event error:", scanErr)
			continue
		}
		_ = json.Unmarshal(rundownBytes, &eventItem.Rundown)
		eventItem.Categories = []models.TicketCategory{} // non-nil empty slice
		eventList = append(eventList, eventItem)
	}

	// Batch loading categories in a single query filtered by current event IDs to eliminate N+1 overhead
	if len(eventList) > 0 {
		eventIDs := make([]string, len(eventList))
		for i, eventItem := range eventList {
			eventIDs[i] = eventItem.ID
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
			for i := range eventList {
				if cats, exists := catMap[eventList[i].ID]; exists {
					eventList[i].Categories = cats
				}
			}
		}
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil data konser", eventList)
}

// GetEventByID godoc
// @Summary Mengambil detail konser berdasarkan ID
// @Description Mengambil detail spesifik dari satu konser berdasarkan ID.
// @Tags Public - Events
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.APIResponse{data=models.EventItem} "Berhasil mengambil detail konser"
// @Failure 404 {object} models.APIResponse "Konser tidak ditemukan"
// @Failure 500 {object} models.APIResponse "Gagal mengambil detail konser"
// @Router /events/{id} [get]
func GetEventByID(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	var eventItem models.EventItem
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
	`, eventID).Scan(&eventItem.ID, &eventItem.Title, &eventItem.Artist, &eventItem.Venue, &eventItem.Date, &eventItem.Time, &eventItem.Category, &eventItem.CategoryBadgeColor, &eventItem.Image, &eventItem.AudioURL, &eventItem.Conductor, &eventItem.OpenGate, &eventItem.Address, &eventItem.Organizer, &eventItem.Subtitle, &rundownBytes, &eventItem.Description, &eventItem.IsClosed)
	_ = json.Unmarshal(rundownBytes, &eventItem.Rundown)

	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ResponseNotFound(ctx, "Konser tidak ditemukan")
		}
		return utils.ResponseInternalError(ctx, "Gagal mengambil detail konser", err)
	}

	eventItem.Categories = []models.TicketCategory{}
	catRows, err := database.DB.Query(`
		SELECT id, event_id, name, price, quota, remaining_quota, created_at
		FROM ticket_categories
		WHERE event_id = $1
		ORDER BY price DESC
	`, eventItem.ID)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var cat models.TicketCategory
			if scanErr := catRows.Scan(&cat.ID, &cat.EventID, &cat.Name, &cat.Price, &cat.Quota, &cat.RemainingQuota, &cat.CreatedAt); scanErr == nil {
				eventItem.Categories = append(eventItem.Categories, cat)
			}
		}
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil detail konser", eventItem)
}

// CreateOrder godoc
// @Summary Membeli tiket / Membuat pesanan
// @Description Membuat pesanan tiket baru untuk event tertentu baik sebagai Guest maupun Logged-in User.
// @Tags Public - Orders
// @Accept json
// @Produce json
// @Param payload body models.CreateOrderRequest true "Order Request Payload"
// @Success 201 {object} models.APIResponse{data=models.OrderRecord} "Pemesanan tiket berhasil!"
// @Failure 400 {object} models.APIResponse "Payload tidak valid / Quota habis"
// @Failure 500 {object} models.APIResponse "Gagal memproses pemesanan"
// @Router /orders [post]
// CreateOrder godoc
// @Summary Membeli tiket / Membuat pesanan
// @Description Membuat pesanan tiket baru untuk event tertentu baik sebagai Guest maupun Logged-in User.
// @Tags Public - Orders
// @Accept json
// @Produce json
// @Param payload body models.CreateOrderRequest true "Order Request Payload"
// @Success 201 {object} models.APIResponse{data=models.OrderRecord} "Pemesanan tiket berhasil!"
// @Failure 400 {object} models.APIResponse "Payload tidak valid / Quota habis"
// @Failure 500 {object} models.APIResponse "Gagal memproses pemesanan"
// @Router /orders [post]
func CreateOrder(ctx *fiber.Ctx) error {
	var createOrderReq models.CreateOrderRequest
	if err := ctx.BodyParser(&createOrderReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	var authenticatedUserID string
	rawAuthHeader := ctx.Get("Authorization")
	if rawAuthHeader != "" && strings.HasPrefix(rawAuthHeader, "Bearer ") {
		rawBearerToken := strings.TrimPrefix(rawAuthHeader, "Bearer ")
		claims, err := utils.ValidateUserToken(rawBearerToken)
		if err == nil && claims != nil {
			authenticatedUserID = claims.UserID
			if createOrderReq.UserEmail == "" {
				createOrderReq.UserEmail = claims.Email
			}
			if createOrderReq.UserName == "" {
				_ = database.DB.QueryRow("SELECT name FROM users WHERE id = $1", authenticatedUserID).Scan(&createOrderReq.UserName)
			}
		}
	}

	if createOrderReq.UserName == "" || createOrderReq.UserEmail == "" || createOrderReq.TicketCategoryID == "" {
		return utils.ResponseBadRequest(ctx, "Nama, Email, dan Kategori Tiket wajib diisi")
	}

	if createOrderReq.Quantity < 1 || createOrderReq.Quantity > 4 {
		return utils.ResponseBadRequest(ctx, "Maksimal pemesanan adalah 1 hingga 4 tiket per transaksi")
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal memulai transaksi basis data", err)
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
	`, createOrderReq.TicketCategoryID).Scan(&catID, &eventID, &catName, &price, &quota, &remainingQuota)

	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ResponseNotFound(ctx, "Kategori tiket tidak ditemukan")
		}
		return utils.ResponseInternalError(ctx, "Gagal memverifikasi kuota tiket", err)
	}

	if remainingQuota < createOrderReq.Quantity {
		return utils.ResponseBadRequest(ctx, fmt.Sprintf("Kuota tiket tidak mencukupi (sisa kuota: %d)", remainingQuota))
	}

	var evtTitle, evtArtist, evtVenue, evtDate, evtTime string
	var evtClosed bool
	err = tx.QueryRow(`
		SELECT title, artist, venue, date, time, is_closed
		FROM events
		WHERE id = $1
	`, eventID).Scan(&evtTitle, &evtArtist, &evtVenue, &evtDate, &evtTime, &evtClosed)

	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengambil data event", err)
	}

	if evtClosed {
		return utils.ResponseBadRequest(ctx, "Penjualan tiket untuk pertunjukan ini telah ditutup karena konser sudah dimulai.")
	}

	// Potong kuota secara atomic
	newRemaining := remainingQuota - createOrderReq.Quantity
	_, err = tx.Exec(`
		UPDATE ticket_categories
		SET remaining_quota = $1
		WHERE id = $2
	`, newRemaining, catID)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal memperbarui kuota tiket", err)
	}

	// Generate kode pesanan & QR Code secara kriptografis aman
	randomNum, err := utils.GenerateCryptoOTP()
	if err != nil {
		randomNum = fmt.Sprintf("%d", time.Now().UnixNano()%900000+100000)
	}
	orderCode := "SYM-" + randomNum
	orderID := uuid.New().String()
	qrCode := fmt.Sprintf("QR-%s", orderCode)
	totalPrice := price * float64(createOrderReq.Quantity)
	dateFull := fmt.Sprintf("%s @ %s", evtDate, evtTime)

	// Simpan transaksi (Status awal: ISSUED)
	_, err = tx.Exec(`
		INSERT INTO orders (id, order_code, user_id, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'ISSUED', 'SANDBOX_PAYMENT')
	`, orderID, orderCode, authenticatedUserID, eventID, evtTitle, evtArtist, evtVenue, dateFull, catName, createOrderReq.Quantity, totalPrice, createOrderReq.UserName, createOrderReq.UserEmail, qrCode)

	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal membuat pesanan tiket", err)
	}

	if err := tx.Commit(); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal konfirmasi transaksi",
		})
	}

	createdOrder := models.OrderRecord{
		ID:            orderID,
		OrderCode:     orderCode,
		UserID:        authenticatedUserID,
		EventID:       eventID,
		EventTitle:    evtTitle,
		Artist:        evtArtist,
		Venue:         evtVenue,
		Date:          dateFull,
		CategoryName:  catName,
		Quantity:      createOrderReq.Quantity,
		TotalPrice:    totalPrice,
		UserName:      createOrderReq.UserName,
		UserEmail:     createOrderReq.UserEmail,
		QRCode:        qrCode,
		Status:        "VERIFIED",
		PaymentMethod: "SANDBOX_PAYMENT",
		CreatedAt:     time.Now(),
	}

	// Kirim E-Ticket secara asinkron ke Mailpit SMTP
	services.SendETicketEmail(createdOrder)

	return utils.ResponseCreated(ctx, "Simulasi Pembayaran Sandbox Berhasil & E-Ticket Terbit!", createdOrder)
}

// LookupTicketByCode godoc
// @Summary Lookup tiket berdasarkan kode pesanan
// @Description Mencari detail tiket E-Ticket berdasarkan order code (tanpa login).
// @Tags Public - Tickets
// @Accept json
// @Produce json
// @Param code query string true "Order Code (misal: SYM-XXXXXX)"
// @Success 200 {object} models.APIResponse{data=models.OrderRecord} "Tiket ditemukan"
// @Failure 400 {object} models.APIResponse "Parameter code wajib diisi"
// @Failure 404 {object} models.APIResponse "Tiket tidak ditemukan"
// @Router /tickets/lookup [get]
func LookupTicketByCode(ctx *fiber.Ctx) error {
	ticketOrderCode := ctx.Query("code")
	if ticketOrderCode == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Query parameter 'code' wajib diisi",
		})
	}

	var foundOrder models.OrderRecord
	err := database.DB.QueryRow(`
		SELECT id, order_code, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method, created_at
		FROM orders
		WHERE LOWER(order_code) = LOWER($1)
	`, ticketOrderCode).Scan(&foundOrder.ID, &foundOrder.OrderCode, &foundOrder.EventID, &foundOrder.EventTitle, &foundOrder.Artist, &foundOrder.Venue, &foundOrder.Date, &foundOrder.CategoryName, &foundOrder.Quantity, &foundOrder.TotalPrice, &foundOrder.UserName, &foundOrder.UserEmail, &foundOrder.QRCode, &foundOrder.Status, &foundOrder.PaymentMethod, &foundOrder.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kode pesanan tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal melakukan pencarian tiket",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Tiket ditemukan",
		Data:    foundOrder,
	})
}

// POST /api/v1/admin/login AdminLogin godoc
// @Summary Login Administrator
// @Description Autentikasi khusus akun administrator untuk mengakses panel manajemen admin.
// @Tags Auth - Admin
// @Accept json
// @Produce json
// @Param payload body models.AdminLoginRequest true "Admin Login Payload"
// @Success 200 {object} models.APIResponse{data=models.AuthResponseData} "Login Admin berhasil"
// @Failure 401 {object} models.APIResponse "Username atau Password Admin salah"
// @Router /admin/login [post]
func AdminLogin(ctx *fiber.Ctx) error {
	var adminLoginReq models.AdminLoginRequest
	if err := ctx.BodyParser(&adminLoginReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
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

	if adminLoginReq.Username != adminUser || adminLoginReq.Password != adminPass {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Username atau Password Admin salah",
		})
	}

	adminJwtToken, err := utils.GenerateUserToken("admin-1", adminUser+"@symphoniatic.id", "ADMIN")
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal menerbitkan token admin", err)
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Login Admin berhasil",
		Data: models.AuthResponseData{
			Token: adminJwtToken,
			User: models.UserRecord{
				ID:         "admin-1",
				Email:      adminUser + "@symphoniatic.id",
				Name:       "Super Admin",
				Role:       "ADMIN",
				IsVerified: true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		},
	})
}

// CreateEvent godoc
// @Summary Membuat event/konser baru (Admin)
// @Description Menambahkan event konser baru ke dalam sistem beserta kategori tiket awal.
// @Tags Admin - Events Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body models.CreateEventRequest true "Create Event Payload"
// @Success 201 {object} models.APIResponse{data=models.EventItem} "Berhasil menambahkan konser baru"
// @Failure 400 {object} models.APIResponse "Judul, Artis, Venue, Tanggal wajib diisi"
// @Router /admin/events [post]
func CreateEvent(ctx *fiber.Ctx) error {
	var createEventReq models.CreateEventRequest
	if err := ctx.BodyParser(&createEventReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if createEventReq.Title == "" || createEventReq.Artist == "" || createEventReq.Venue == "" || createEventReq.Date == "" || createEventReq.Time == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Judul, Artist, Venue, Tanggal, dan Waktu wajib diisi",
		})
	}

	if createEventReq.Category == "" {
		createEventReq.Category = "SIMFONI UTAMA"
	}
	if createEventReq.CategoryBadgeColor == "" {
		createEventReq.CategoryBadgeColor = "bg-blue-900/80 text-blue-200 border-blue-500/40"
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memulai transaksi",
		})
	}
	defer tx.Rollback()

	eventID := fmt.Sprintf("evt-%s", uuid.New().String()[:8])
	rundownJSON, _ := json.Marshal(createEventReq.Rundown)
	if createEventReq.Rundown == nil {
		rundownJSON = []byte("[]")
	}
	_, err = tx.Exec(`
		INSERT INTO events (id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, conductor, open_gate, address, organizer, subtitle, rundown, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, eventID, createEventReq.Title, createEventReq.Artist, createEventReq.Venue, createEventReq.Date, createEventReq.Time, createEventReq.Category, createEventReq.CategoryBadgeColor, createEventReq.Image, createEventReq.AudioURL, createEventReq.Conductor, createEventReq.OpenGate, createEventReq.Address, createEventReq.Organizer, createEventReq.Subtitle, string(rundownJSON), createEventReq.Description)

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menyimpan event konser ke database",
			Error:   err.Error(),
		})
	}

	// Insert categories if provided
	var createdCategories []models.TicketCategory
	for idx, catInput := range createEventReq.Categories {
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
			return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Message: "Gagal menyimpan kategori tiket",
				Error:   err.Error(),
			})
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
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal konfirmasi transaksi event",
		})
	}

	createdEventItem := models.EventItem{
		ID:                 eventID,
		Title:              createEventReq.Title,
		Artist:             createEventReq.Artist,
		Venue:              createEventReq.Venue,
		Date:               createEventReq.Date,
		Time:               createEventReq.Time,
		Category:           createEventReq.Category,
		CategoryBadgeColor: createEventReq.CategoryBadgeColor,
		Image:              createEventReq.Image,
		AudioURL:           createEventReq.AudioURL,
		Conductor:          createEventReq.Conductor,
		OpenGate:           createEventReq.OpenGate,
		Address:            createEventReq.Address,
		Organizer:          createEventReq.Organizer,
		Subtitle:           createEventReq.Subtitle,
		Description:        createEventReq.Description,
		Rundown:            createEventReq.Rundown,
		Categories:         createdCategories,
	}

	return ctx.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Event konser berhasil ditambahkan",
		Data:    createdEventItem,
	})
}

// PUT /api/v1/admin/events/:id (Update Event)
func UpdateEvent(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	var updateEventReq models.UpdateEventRequest
	if err := ctx.BodyParser(&updateEventReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	rundownJSON, _ := json.Marshal(updateEventReq.Rundown)
	if updateEventReq.Rundown == nil {
		rundownJSON = []byte("[]")
	}
	updateResult, err := database.DB.Exec(`
		UPDATE events
		SET title = $1, artist = $2, venue = $3, date = $4, time = $5, category = $6, category_badge_color = $7, image = $8, audio_url = $9, conductor = $10, open_gate = $11, address = $12, organizer = $13, subtitle = $14, rundown = $15, description = $16
		WHERE id = $17
	`, updateEventReq.Title, updateEventReq.Artist, updateEventReq.Venue, updateEventReq.Date, updateEventReq.Time, updateEventReq.Category, updateEventReq.CategoryBadgeColor, updateEventReq.Image, updateEventReq.AudioURL, updateEventReq.Conductor, updateEventReq.OpenGate, updateEventReq.Address, updateEventReq.Organizer, updateEventReq.Subtitle, string(rundownJSON), updateEventReq.Description, eventID)

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui event konser",
			Error:   err.Error(),
		})
	}

	affectedRows, _ := updateResult.RowsAffected()
	if affectedRows == 0 {
		return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Event konser tidak ditemukan",
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Event konser berhasil diperbarui",
	})
}

// DELETE /api/v1/admin/events/:id (Delete Event)
func DeleteEvent(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	deleteResult, err := database.DB.Exec("DELETE FROM events WHERE id = $1", eventID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menghapus event konser",
			Error:   err.Error(),
		})
	}

	affectedRows, _ := deleteResult.RowsAffected()
	if affectedRows == 0 {
		return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Event konser tidak ditemukan",
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Event konser dan kategori tiket terkait berhasil dihapus",
	})
}

// POST /api/v1/admin/events/:id/categories (Add Ticket Category)
func CreateTicketCategory(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	var createCategoryReq models.CreateCategoryInput
	if err := ctx.BodyParser(&createCategoryReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if createCategoryReq.Name == "" || createCategoryReq.Price <= 0 || createCategoryReq.Quota <= 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Nama, Harga, dan Kuota tiket wajib valid",
		})
	}

	catID := fmt.Sprintf("cat-%s-%s", eventID, uuid.New().String()[:6])
	_, err := database.DB.Exec(`
		INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, catID, eventID, createCategoryReq.Name, createCategoryReq.Price, createCategoryReq.Quota, createCategoryReq.Quota)

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menambah kategori tiket",
			Error:   err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil ditambahkan",
		Data: models.TicketCategory{
			ID:             catID,
			EventID:        eventID,
			Name:           createCategoryReq.Name,
			Price:          createCategoryReq.Price,
			Quota:          createCategoryReq.Quota,
			RemainingQuota: createCategoryReq.Quota,
			CreatedAt:      time.Now(),
		},
	})
}

// PUT /api/v1/admin/categories/:id (Update Ticket Category)
func UpdateTicketCategory(ctx *fiber.Ctx) error {
	catID := ctx.Params("id")
	var updateCategoryReq models.UpdateTicketCategoryRequest
	if err := ctx.BodyParser(&updateCategoryReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	updateResult, err := database.DB.Exec(`
		UPDATE ticket_categories
		SET name = $1, price = $2, remaining_quota = GREATEST(0, remaining_quota + ($3 - quota)), quota = $3
		WHERE id = $4
	`, updateCategoryReq.Name, updateCategoryReq.Price, updateCategoryReq.Quota, catID)

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui kategori tiket",
			Error:   err.Error(),
		})
	}

	affectedRows, _ := updateResult.RowsAffected()
	if affectedRows == 0 {
		return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Kategori tiket tidak ditemukan",
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil diperbarui",
	})
}

// DELETE /api/v1/admin/categories/:id (Delete Category)
func DeleteTicketCategory(ctx *fiber.Ctx) error {
	catID := ctx.Params("id")
	deleteResult, err := database.DB.Exec("DELETE FROM ticket_categories WHERE id = $1", catID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menghapus kategori tiket",
			Error:   err.Error(),
		})
	}

	affectedRows, _ := deleteResult.RowsAffected()
	if affectedRows == 0 {
		return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Kategori tiket tidak ditemukan",
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil dihapus",
	})
}

// GET /api/v1/admin/orders (List all orders with optional filter & search)
func GetAllOrders(ctx *fiber.Ctx) error {
	searchQuery := ctx.Query("search")
	statusFilter := ctx.Query("status")

	query := `
		SELECT id, order_code, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method, created_at
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

	orderRows, err := database.DB.Query(query, queryArgs...)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil daftar pesanan",
			Error:   err.Error(),
		})
	}
	defer orderRows.Close()

	var allOrders []models.OrderRecord
	for orderRows.Next() {
		var orderRecord models.OrderRecord
		err := orderRows.Scan(&orderRecord.ID, &orderRecord.OrderCode, &orderRecord.EventID, &orderRecord.EventTitle, &orderRecord.Artist, &orderRecord.Venue, &orderRecord.Date, &orderRecord.CategoryName, &orderRecord.Quantity, &orderRecord.TotalPrice, &orderRecord.UserName, &orderRecord.UserEmail, &orderRecord.QRCode, &orderRecord.Status, &orderRecord.PaymentMethod, &orderRecord.CreatedAt)
		if err == nil {
			allOrders = append(allOrders, orderRecord)
		}
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Daftar pesanan berhasil diambil",
		Data:    allOrders,
	})
}

// PATCH /api/v1/admin/orders/:id/status (Update Order Status)
func UpdateOrderStatus(ctx *fiber.Ctx) error {
	orderID := ctx.Params("id")
	var updateStatusReq models.UpdateOrderStatusRequest
	if err := ctx.BodyParser(&updateStatusReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if updateStatusReq.Status == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Status pesanan wajib diisi",
		})
	}

	updateResult, err := database.DB.Exec("UPDATE orders SET status = $1 WHERE id = $2", strings.ToUpper(updateStatusReq.Status), orderID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status pesanan",
			Error:   err.Error(),
		})
	}

	affectedRows, _ := updateResult.RowsAffected()
	if affectedRows == 0 {
		return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Pesanan tidak ditemukan",
		})
	}

	if strings.ToUpper(updateStatusReq.Status) == "REMINDED" || strings.ToUpper(updateStatusReq.Status) == "REMINDER" {
		var userEmail, userName, eventTitle, venue, date, orderCode string
		err := database.DB.QueryRow("SELECT user_email, user_name, event_title, venue, date, order_code FROM orders WHERE id = $1", orderID).Scan(&userEmail, &userName, &eventTitle, &venue, &date, &orderCode)
		if err == nil {
			services.SendEventReminderEmail(userEmail, userName, eventTitle, venue, "", date, orderCode)
		}
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Status pesanan berhasil diperbarui menjadi %s", updateStatusReq.Status),
	})
}

// GET /api/v1/admin/dashboard (Enhanced Admin Dashboard Analytics)
func GetAdminDashboardMetrics(ctx *fiber.Ctx) error {
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
	eventRevenueRows, err := database.DB.Query(`
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
		defer eventRevenueRows.Close()
		for eventRevenueRows.Next() {
			var eventRevenueItem EventRevenueStat
			if scanErr := eventRevenueRows.Scan(&eventRevenueItem.EventID, &eventRevenueItem.Title, &eventRevenueItem.Revenue, &eventRevenueItem.TicketsSold); scanErr == nil {
				eventStats = append(eventStats, eventRevenueItem)
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
	currentTime := time.Now()
	for i := 5; i >= 0; i-- {
		targetMonthTime := currentTime.AddDate(0, -i, 0)
		revenueTimeline[5-i] = TimelineStat{
			Month:   targetMonthTime.Format("Jan"),
			Revenue: 0,
			Tickets: 0,
		}
	}

	if err == nil {
		defer timelineRows.Close()
		for timelineRows.Next() {
			var timelineItem TimelineStat
			if scanErr := timelineRows.Scan(&timelineItem.Month, &timelineItem.Revenue, &timelineItem.Tickets); scanErr == nil {
				// Overwrite matching month
				for idx, item := range revenueTimeline {
					if strings.EqualFold(item.Month, timelineItem.Month) {
						revenueTimeline[idx].Revenue = timelineItem.Revenue
						revenueTimeline[idx].Tickets = timelineItem.Tickets
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
			var categoryItem CategoryStat
			if scanErr := catRows.Scan(&categoryItem.Name, &categoryItem.Value); scanErr == nil {
				categoryDistribution = append(categoryDistribution, categoryItem)
			}
		}
	}

	// Recent 5 Orders
	recentOrderRows, err := database.DB.Query(`
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
		defer recentOrderRows.Close()
		for recentOrderRows.Next() {
			var recentOrderItem RecentOrderSummary
			if scanErr := recentOrderRows.Scan(&recentOrderItem.ID, &recentOrderItem.OrderCode, &recentOrderItem.EventTitle, &recentOrderItem.Quantity, &recentOrderItem.TotalPrice, &recentOrderItem.UserName, &recentOrderItem.Status, &recentOrderItem.CreatedAt); scanErr == nil {
				recentOrders = append(recentOrders, recentOrderItem)
			}
		}
	}

	return ctx.JSON(models.APIResponse{
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
func RequestRefundOTP(ctx *fiber.Ctx) error {
	var requestOTPInput models.RequestRefundOTPInput
	if err := ctx.BodyParser(&requestOTPInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
			Error:   err.Error(),
		})
	}

	orderCode := strings.TrimSpace(requestOTPInput.OrderCode)
	userEmail := strings.ToLower(strings.TrimSpace(requestOTPInput.UserEmail))

	if orderCode == "" || userEmail == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode pesanan dan email wajib diisi",
		})
	}

	// 1. Check order existence and ownership
	var targetOrder models.OrderRecord
	err := database.DB.QueryRow(`
		SELECT id, order_code, event_title, user_email, status, total_price, quantity
		FROM orders
		WHERE UPPER(order_code) = UPPER($1)
	`, orderCode).Scan(&targetOrder.ID, &targetOrder.OrderCode, &targetOrder.EventTitle, &targetOrder.UserEmail, &targetOrder.Status, &targetOrder.TotalPrice, &targetOrder.Quantity)

	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kode pesanan tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memverifikasi pesanan",
			Error:   err.Error(),
		})
	}

	// Email check
	if !strings.EqualFold(targetOrder.UserEmail, userEmail) {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Email tidak cocok dengan pemegang tiket ini",
		})
	}

	// Check order status
	if targetOrder.Status == "CHECKED_IN" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Tiket sudah di-scan (Check In) pada venue dan tidak dapat dikembalikan / refund",
		})
	}
	if targetOrder.Status == "REFUND_REQUESTED" || targetOrder.Status == "REFUNDED" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Pengajuan refund untuk tiket ini sedang diproses atau sudah selesai",
		})
	}
	if targetOrder.Status == "CANCELLED" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Pesanan ini sudah dibatalkan sebelumnya",
		})
	}

	// Generate 6-digit OTP
	otpCode := fmt.Sprintf("%06d", rand.Intn(1000000))
	expiresAt := time.Now().Add(10 * time.Minute)

	// Check if existing refund_request record exists
	var existingRefundID string
	err = database.DB.QueryRow(`
		SELECT id FROM refund_requests WHERE order_id = $1
	`, targetOrder.ID).Scan(&existingRefundID)

	if err == sql.ErrNoRows {
		refundID := "rf-" + uuid.New().String()
		_, err = database.DB.Exec(`
			INSERT INTO refund_requests (id, order_id, order_code, user_email, bank_name, account_number, account_holder, reason, refund_amount, status, otp_code, otp_expires_at)
			VALUES ($1, $2, $3, $4, '', '', '', '', $5, 'OTP_SENT', $6, $7)
		`, refundID, targetOrder.ID, targetOrder.OrderCode, targetOrder.UserEmail, targetOrder.TotalPrice, otpCode, expiresAt)
	} else if err == nil {
		_, err = database.DB.Exec(`
			UPDATE refund_requests
			SET otp_code = $1, otp_expires_at = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`, otpCode, expiresAt, existingRefundID)
	}

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat kode verifikasi OTP",
			Error:   err.Error(),
		})
	}

	// Send OTP email
	services.SendRefundOTPEmail(targetOrder.UserEmail, targetOrder.OrderCode, otpCode)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Kode OTP verifikasi refund telah dikirim ke email " + targetOrder.UserEmail,
		Data: fiber.Map{
			"orderCode": targetOrder.OrderCode,
			"userEmail": targetOrder.UserEmail,
			"expiresIn": "10 Menit",
		},
	})
}

// POST /api/v1/refunds/submit
func SubmitRefund(ctx *fiber.Ctx) error {
	var submitRefundInput models.SubmitRefundInput
	if err := ctx.BodyParser(&submitRefundInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
			Error:   err.Error(),
		})
	}

	orderCode := strings.TrimSpace(submitRefundInput.OrderCode)
	userEmail := strings.ToLower(strings.TrimSpace(submitRefundInput.UserEmail))
	otpCode := strings.TrimSpace(submitRefundInput.OTPCode)
	bankName := strings.TrimSpace(submitRefundInput.BankName)
	accountNumber := strings.TrimSpace(submitRefundInput.AccountNumber)
	accountHolder := strings.TrimSpace(submitRefundInput.AccountHolder)
	reason := strings.TrimSpace(submitRefundInput.Reason)

	if orderCode == "" || userEmail == "" || otpCode == "" || bankName == "" || accountNumber == "" || accountHolder == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Seluruh kolom formulir dan OTP wajib diisi",
		})
	}

	// Query refund_request record
	var refundRecord models.RefundRequestRecord
	var storedOTPCode string
	var otpExpiresAt time.Time
	var currentRefundStatus string

	err := database.DB.QueryRow(`
		SELECT id, order_id, order_code, user_email, status, otp_code, otp_expires_at
		FROM refund_requests
		WHERE UPPER(order_code) = UPPER($1)
	`, orderCode).Scan(&refundRecord.ID, &refundRecord.OrderID, &refundRecord.OrderCode, &refundRecord.UserEmail, &currentRefundStatus, &storedOTPCode, &otpExpiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Pengajuan refund tidak ditemukan. Silakan minta kode OTP terlebih dahulu.",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memproses data refund",
			Error:   err.Error(),
		})
	}

	if !strings.EqualFold(refundRecord.UserEmail, userEmail) {
		return ctx.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Email tidak sesuai dengan pemegang tiket ini",
		})
	}

	if storedOTPCode != otpCode {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode OTP yang Anda masukkan salah",
		})
	}

	if time.Now().After(otpExpiresAt) {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode OTP telah kadaluarsa. Silakan minta kode OTP baru.",
		})
	}

	// Start SQL Transaction
	tx, err := database.DB.Begin()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
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
	`, bankName, accountNumber, accountHolder, reason, refundRecord.ID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
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
	`, refundRecord.OrderID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status pesanan",
			Error:   err.Error(),
		})
	}

	if err := tx.Commit(); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menyimpan pengajuan refund",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Pengajuan refund berhasil dikirim. Tim Finance akan meninjau dan memproses pengembalian dana Anda.",
		Data: fiber.Map{
			"orderCode":     refundRecord.OrderCode,
			"status":        "PENDING",
			"bankName":      bankName,
			"accountHolder": accountHolder,
		},
	})
}

// POST /api/v1/refunds/status
func GetRefundStatus(ctx *fiber.Ctx) error {
	var checkStatusInput models.CheckRefundStatusRequest
	if err := ctx.BodyParser(&checkStatusInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
		})
	}

	orderCode := strings.TrimSpace(checkStatusInput.OrderCode)
	userEmail := strings.ToLower(strings.TrimSpace(checkStatusInput.UserEmail))

	if orderCode == "" || userEmail == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Kode pesanan dan email wajib diisi",
		})
	}

	var refundRecord models.RefundRequestRecord
	var orderStatus string

	err := database.DB.QueryRow(`
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, r.account_holder, r.reason, r.refund_amount, r.status, COALESCE(r.admin_note, ''), r.created_at, r.updated_at, o.status, o.event_title, o.category_name, o.quantity, o.user_name
		FROM refund_requests r
		JOIN orders o ON o.id = r.order_id
		WHERE UPPER(r.order_code) = UPPER($1) AND LOWER(r.user_email) = LOWER($2)
	`, orderCode, userEmail).Scan(
		&refundRecord.ID, &refundRecord.OrderID, &refundRecord.OrderCode, &refundRecord.UserEmail, &refundRecord.BankName, &refundRecord.AccountNumber, &refundRecord.AccountHolder,
		&refundRecord.Reason, &refundRecord.RefundAmount, &refundRecord.Status, &refundRecord.AdminNote, &refundRecord.CreatedAt, &refundRecord.UpdatedAt,
		&orderStatus, &refundRecord.EventTitle, &refundRecord.CategoryName, &refundRecord.Quantity, &refundRecord.UserName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Data pengajuan refund tidak ditemukan untuk kode dan email ini",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil status refund",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil mengambil status refund",
		Data: fiber.Map{
			"orderStatus":  orderStatus,
			"refundDetail": refundRecord,
		},
	})
}

// GET /api/v1/admin/refunds
func AdminGetAllRefunds(ctx *fiber.Ctx) error {
	refundRows, err := database.DB.Query(`
		SELECT r.id, r.order_id, r.order_code, r.user_email, r.bank_name, r.account_number, r.account_holder, r.reason, r.refund_amount, r.status, COALESCE(r.admin_note, ''), r.created_at, r.updated_at, o.event_title, o.category_name, o.quantity, o.user_name
		FROM refund_requests r
		JOIN orders o ON o.id = r.order_id
		WHERE r.status != 'OTP_SENT'
		ORDER BY r.created_at DESC
	`)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil daftar permohonan refund",
			Error:   err.Error(),
		})
	}
	defer refundRows.Close()

	var refundList []models.RefundRequestRecord
	for refundRows.Next() {
		var refundRecord models.RefundRequestRecord
		err := refundRows.Scan(
			&refundRecord.ID, &refundRecord.OrderID, &refundRecord.OrderCode, &refundRecord.UserEmail, &refundRecord.BankName, &refundRecord.AccountNumber, &refundRecord.AccountHolder,
			&refundRecord.Reason, &refundRecord.RefundAmount, &refundRecord.Status, &refundRecord.AdminNote, &refundRecord.CreatedAt, &refundRecord.UpdatedAt,
			&refundRecord.EventTitle, &refundRecord.CategoryName, &refundRecord.Quantity, &refundRecord.UserName,
		)
		if err == nil {
			refundList = append(refundList, refundRecord)
		}
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Berhasil mengambil daftar permohonan refund",
		Data:    refundList,
	})
}

// PATCH /api/v1/admin/refunds/:id/status
func AdminUpdateRefundStatus(ctx *fiber.Ctx) error {
	refundID := ctx.Params("id")
	var updateRefundStatusReq models.UpdateRefundStatusRequest
	if err := ctx.BodyParser(&updateRefundStatusReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format data request tidak valid",
		})
	}

	newStatus := strings.ToUpper(strings.TrimSpace(updateRefundStatusReq.Status))
	if newStatus != "APPROVED" && newStatus != "REJECTED" && newStatus != "COMPLETED" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
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
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Data refund tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil detail refund",
			Error:   err.Error(),
		})
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
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
	`, newStatus, updateRefundStatusReq.AdminNote, refundID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
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
			return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
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
			return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
				Success: false,
				Message: "Gagal mengembalikan status order",
				Error:   err.Error(),
			})
		}
	}

	if err := tx.Commit(); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal melakukan commit transaksi",
			Error:   err.Error(),
		})
	}

	// Send status email notification
	services.SendRefundStatusNotificationEmail(userEmail, orderCode, newStatus, updateRefundStatusReq.AdminNote, amount)

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Status pengajuan refund berhasil diubah menjadi %s", newStatus),
	})
}

// PATCH /api/v1/admin/events/:id/toggle-close
func ToggleEventClose(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	var isClosed bool
	err := database.DB.QueryRow(`SELECT is_closed FROM events WHERE id = $1`, eventID).Scan(&isClosed)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Konser tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengecek status konser",
			Error:   err.Error(),
		})
	}

	newClosedState := !isClosed
	_, err = database.DB.Exec(`UPDATE events SET is_closed = $1 WHERE id = $2`, newClosedState, eventID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status penutupan konser",
			Error:   err.Error(),
		})
	}

	msg := "Penjualan tiket konser berhasil ditutup"
	if !newClosedState {
		msg = "Penjualan tiket konser berhasil dibuka kembali"
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: msg,
		Data: fiber.Map{
			"eventId":  eventID,
			"isClosed": newClosedState,
		},
	})
}

// POST /api/v1/admin/upload
func UploadImage(ctx *fiber.Ctx) error {
	uploadedFile, err := ctx.FormFile("image")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membaca file gambar",
			Error:   err.Error(),
		})
	}

	// Create uploads directory if not exists
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal membuat folder penyimpanan",
			Error:   err.Error(),
		})
	}

	// Unique filename
	fileExtension := filepath.Ext(uploadedFile.Filename)
	uniqueFilename := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExtension)
	uploadDestPath := fmt.Sprintf("./uploads/%s", uniqueFilename)

	if err := ctx.SaveFile(uploadedFile, uploadDestPath); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
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

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "File gambar berhasil diunggah",
		Data: fiber.Map{
			"url": fmt.Sprintf("%s/uploads/%s", baseURL, uniqueFilename),
		},
	})
}



