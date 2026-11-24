package database

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	_ "github.com/lib/pq"
)

var DB *sql.DB

func TryConnectDBWithConfig(cfg *config.DatabaseConfig) error {
	var err error
	DB, err = sql.Open("postgres", cfg.DSN())
	if err != nil {
		return err
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute)

	if err = DB.Ping(); err != nil {
		return err
	}

	log.Println("Berhasil terhubung ke database PostgreSQL")
	initSchema()
	return nil
}

func TryConnectDB() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	return TryConnectDBWithConfig(&cfg.Database)
}

func ConnectDBWithConfig(cfg *config.DatabaseConfig) {
	if err := TryConnectDBWithConfig(cfg); err != nil {
		log.Fatalf("Gagal membuka koneksi PostgreSQL: %v", err)
	}
}

func ConnectDB() {
	if err := TryConnectDB(); err != nil {
		log.Fatalf("Gagal membuka koneksi PostgreSQL: %v", err)
	}
}

func initSchema() {
	schemaQuery := `
	CREATE TABLE IF NOT EXISTS events (
		id VARCHAR(64) PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		artist VARCHAR(255) NOT NULL,
		venue VARCHAR(255) NOT NULL,
		date VARCHAR(100) NOT NULL,
		time VARCHAR(50) NOT NULL,
		category VARCHAR(100) NOT NULL,
		category_badge_color VARCHAR(255) NOT NULL,
		image TEXT NOT NULL,
		audio_url TEXT NOT NULL,
		description TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ticket_categories (
		id VARCHAR(64) PRIMARY KEY,
		event_id VARCHAR(64) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL,
		price NUMERIC(12, 2) NOT NULL,
		quota INT NOT NULL,
		remaining_quota INT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS orders (
		id VARCHAR(64) PRIMARY KEY,
		order_code VARCHAR(100) UNIQUE NOT NULL,
		event_id VARCHAR(64) NOT NULL REFERENCES events(id),
		event_title VARCHAR(255) NOT NULL,
		artist VARCHAR(255) NOT NULL,
		venue VARCHAR(255) NOT NULL,
		date VARCHAR(100) NOT NULL,
		category_name VARCHAR(100) NOT NULL,
		quantity INT NOT NULL,
		total_price NUMERIC(12, 2) NOT NULL,
		user_name VARCHAR(255) NOT NULL,
		user_email VARCHAR(255) NOT NULL,
		qr_code VARCHAR(255) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'ISSUED',
		payment_method VARCHAR(50) DEFAULT 'SANDBOX_PAYMENT',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	ALTER TABLE events ADD COLUMN IF NOT EXISTS conductor VARCHAR(255) DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS open_gate VARCHAR(100) DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS address TEXT DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS organizer VARCHAR(255) DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS subtitle VARCHAR(255) DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS rundown JSONB DEFAULT '[]';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS is_closed BOOLEAN DEFAULT FALSE;
	ALTER TABLE events ADD COLUMN IF NOT EXISTS event_date_time TIMESTAMP WITH TIME ZONE DEFAULT NULL;
	ALTER TABLE events ADD COLUMN IF NOT EXISTS sales_end_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
	ALTER TABLE events ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS cancellation_reason TEXT NOT NULL DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS reschedule_reason TEXT NOT NULL DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS original_date VARCHAR(100) NOT NULL DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS original_time VARCHAR(50) NOT NULL DEFAULT '';
	ALTER TABLE events ADD COLUMN IF NOT EXISTS status_updated_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;

	ALTER TABLE orders ADD COLUMN IF NOT EXISTS user_id VARCHAR(64) DEFAULT '';
	ALTER TABLE orders ADD COLUMN IF NOT EXISTS promo_code VARCHAR(50) DEFAULT '';
	ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(12, 2) DEFAULT 0;
	ALTER TABLE orders ADD COLUMN IF NOT EXISTS seat_numbers VARCHAR(255) DEFAULT '';
	ALTER TABLE orders ADD COLUMN IF NOT EXISTS attendee_names TEXT DEFAULT '';
	ALTER TABLE orders ADD COLUMN IF NOT EXISTS checked_in_seats VARCHAR(255) DEFAULT '';
	ALTER TABLE orders ADD COLUMN IF NOT EXISTS seat_checkins TEXT DEFAULT '';
	ALTER TABLE orders ADD COLUMN IF NOT EXISTS checked_in_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
	CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
	CREATE INDEX IF NOT EXISTS idx_orders_event_id ON orders(event_id);
	CREATE INDEX IF NOT EXISTS idx_orders_user_email_lower ON orders(LOWER(user_email));
	CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
	CREATE INDEX IF NOT EXISTS idx_orders_created_at_status ON orders(created_at, status);
	CREATE INDEX IF NOT EXISTS idx_ticket_categories_event_id ON ticket_categories(event_id);

	CREATE TABLE IF NOT EXISTS seats (
		id VARCHAR(64) PRIMARY KEY,
		event_id VARCHAR(64) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
		category_id VARCHAR(64) NOT NULL REFERENCES ticket_categories(id) ON DELETE CASCADE,
		seat_code VARCHAR(20) NOT NULL,
		row_label VARCHAR(10) NOT NULL,
		seat_number INT NOT NULL,
		order_id VARCHAR(64) REFERENCES orders(id) ON DELETE SET NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(event_id, seat_code)
	);

	CREATE INDEX IF NOT EXISTS idx_seats_event_id ON seats(event_id);
	CREATE INDEX IF NOT EXISTS idx_seats_category_id ON seats(category_id);
	CREATE INDEX IF NOT EXISTS idx_seats_order_id ON seats(order_id);
	CREATE INDEX IF NOT EXISTS idx_seats_event_status ON seats(event_id, status);

	CREATE TABLE IF NOT EXISTS refund_requests (
		id VARCHAR(64) PRIMARY KEY,
		order_id VARCHAR(64) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		order_code VARCHAR(100) NOT NULL,
		user_email VARCHAR(255) NOT NULL,
		bank_name VARCHAR(100) NOT NULL,
		account_number VARCHAR(100) NOT NULL,
		account_holder VARCHAR(255) NOT NULL,
		reason TEXT DEFAULT '',
		refund_amount NUMERIC(12, 2) NOT NULL,
		status VARCHAR(50) DEFAULT 'PENDING',
		admin_note TEXT DEFAULT '',
		otp_code VARCHAR(10) DEFAULT NULL,
		otp_expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
		attempts INT DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	ALTER TABLE refund_requests ADD COLUMN IF NOT EXISTS attempts INT DEFAULT 0;

	CREATE INDEX IF NOT EXISTS idx_refund_requests_order_id ON refund_requests(order_id);
	CREATE INDEX IF NOT EXISTS idx_refund_requests_order_code ON refund_requests(order_code);
	CREATE INDEX IF NOT EXISTS idx_refund_requests_status ON refund_requests(status);
	CREATE INDEX IF NOT EXISTS idx_refund_requests_user_email_lower ON refund_requests(LOWER(user_email));

	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(64) PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		phone VARCHAR(50) DEFAULT '',
		password_hash VARCHAR(255) DEFAULT '',
		role VARCHAR(50) DEFAULT 'USER',
		is_verified BOOLEAN DEFAULT TRUE,
		email_promo_opt_out BOOLEAN NOT NULL DEFAULT FALSE,
		unsubscribe_token VARCHAR(64) DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	ALTER TABLE users ADD COLUMN IF NOT EXISTS email_promo_opt_out BOOLEAN NOT NULL DEFAULT FALSE;
	ALTER TABLE users ADD COLUMN IF NOT EXISTS unsubscribe_token VARCHAR(64) DEFAULT '';

	CREATE TABLE IF NOT EXISTS auth_otps (
		id VARCHAR(64) PRIMARY KEY,
		email VARCHAR(255) NOT NULL,
		otp_code VARCHAR(64) NOT NULL,
		purpose VARCHAR(50) NOT NULL,
		attempts INT DEFAULT 0,
		is_used BOOLEAN DEFAULT FALSE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_auth_otps_email_purpose ON auth_otps(email, purpose);

	CREATE TABLE IF NOT EXISTS promotions (
		id VARCHAR(64) PRIMARY KEY,
		code VARCHAR(50) UNIQUE NOT NULL,
		description TEXT DEFAULT '',
		discount_type VARCHAR(20) NOT NULL,
		discount_value NUMERIC(12, 2) NOT NULL,
		min_order_amount NUMERIC(12, 2) DEFAULT 0,
		max_discount_amount NUMERIC(12, 2) DEFAULT 0,
		is_member_only BOOLEAN DEFAULT TRUE,
		is_active BOOLEAN DEFAULT TRUE,
		quota INT DEFAULT 100,
		remaining_quota INT DEFAULT 100,
		valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
		valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	ALTER TABLE promotions ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;

	CREATE TABLE IF NOT EXISTS user_promo_usages (
		id VARCHAR(64) PRIMARY KEY,
		promotion_id VARCHAR(64) NOT NULL REFERENCES promotions(id) ON DELETE CASCADE,
		user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		order_id VARCHAR(64) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT unique_user_promo UNIQUE (promotion_id, user_id)
	);

	CREATE INDEX IF NOT EXISTS idx_promotions_code_upper ON promotions(UPPER(code));
	CREATE INDEX IF NOT EXISTS idx_user_promo_usages_user ON user_promo_usages(user_id);

	CREATE TABLE IF NOT EXISTS admin_users (
		id VARCHAR(64) PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		name VARCHAR(100) NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(30) NOT NULL DEFAULT 'ADMIN',
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_admin_users_username_lower ON admin_users(LOWER(username));
	CREATE INDEX IF NOT EXISTS idx_admin_users_role ON admin_users(role);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id VARCHAR(64) PRIMARY KEY,
		actor_id VARCHAR(64) NOT NULL,
		actor_name VARCHAR(100) NOT NULL,
		actor_role VARCHAR(30) NOT NULL,
		action VARCHAR(100) NOT NULL,
		target_resource VARCHAR(150) NOT NULL,
		details TEXT DEFAULT '',
		ip_address VARCHAR(45) DEFAULT '',
		category VARCHAR(30) NOT NULL DEFAULT 'OPERATIONAL',
		severity VARCHAR(20) NOT NULL DEFAULT 'INFO',
		status_code INT NOT NULL DEFAULT 200,
		user_agent TEXT DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS category VARCHAR(30) NOT NULL DEFAULT 'OPERATIONAL';
	ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS severity VARCHAR(20) NOT NULL DEFAULT 'INFO';
	ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS status_code INT NOT NULL DEFAULT 200;
	ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_agent TEXT DEFAULT '';

	CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_role ON audit_logs(actor_role);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_category ON audit_logs(category);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_severity ON audit_logs(severity);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_ip_address ON audit_logs(ip_address);

	CREATE TABLE IF NOT EXISTS blocked_ips (
		id VARCHAR(64) PRIMARY KEY,
		ip_address VARCHAR(45) NOT NULL,
		reason TEXT NOT NULL,
		blocked_by VARCHAR(100) NOT NULL,
		is_auto BOOLEAN NOT NULL DEFAULT FALSE,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		expires_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_blocked_ips_ip_active ON blocked_ips(ip_address, is_active);

	CREATE TABLE IF NOT EXISTS email_broadcasts (
		id VARCHAR(64) PRIMARY KEY,
		target_scope VARCHAR(30) NOT NULL,
		target_event_id VARCHAR(64) REFERENCES events(id) ON DELETE SET NULL,
		target_category_id VARCHAR(64) DEFAULT '',
		category VARCHAR(30) NOT NULL,
		subject VARCHAR(255) NOT NULL,
		headline VARCHAR(255) NOT NULL,
		body_text TEXT NOT NULL,
		cta_label VARCHAR(100) NOT NULL DEFAULT '',
		cta_url TEXT NOT NULL DEFAULT '',
		promo_code VARCHAR(50) NOT NULL DEFAULT '',
		total_recipients INT NOT NULL DEFAULT 0,
		sent_count INT NOT NULL DEFAULT 0,
		failed_count INT NOT NULL DEFAULT 0,
		status VARCHAR(20) NOT NULL DEFAULT 'PROCESSING',
		created_by VARCHAR(100) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP WITH TIME ZONE
	);

	ALTER TABLE email_broadcasts ADD COLUMN IF NOT EXISTS target_category_id VARCHAR(64) DEFAULT '';

	CREATE TABLE IF NOT EXISTS email_broadcast_recipients (
		id VARCHAR(64) PRIMARY KEY,
		broadcast_id VARCHAR(64) NOT NULL REFERENCES email_broadcasts(id) ON DELETE CASCADE,
		recipient_name VARCHAR(255) NOT NULL DEFAULT '',
		recipient_email VARCHAR(255) NOT NULL,
		order_code VARCHAR(100) DEFAULT '',
		status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
		error_message TEXT DEFAULT '',
		sent_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_email_broadcasts_created_at ON email_broadcasts(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_email_broadcasts_target_event ON email_broadcasts(target_event_id);
	CREATE INDEX IF NOT EXISTS idx_broadcast_recipients_bcast_status ON email_broadcast_recipients(broadcast_id, status);
	`

	_, err := DB.Exec(schemaQuery)
	if err != nil {
		log.Fatalf("Gagal inisialisasi skema tabel PostgreSQL: %v", err)
	}

	log.Println("Skema database PostgreSQL (events, ticket_categories, orders, seats, users, auth_otps, promotions, admin_users, audit_logs, blocked_ips) siap")
	seedInitialData()
	ensurePromotionsSeeded()
	ensureAdminUsersSeeded()
	ensureEventSeatsInitialized()
}

func ensureAdminUsersSeeded() {
	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count); err == nil && count > 0 {
		// Tetap pastikan akun superadmin ada jika database lama sudah punya data admin_users tanpa SUPER_ADMIN
		var superCount int
		if err := DB.QueryRow("SELECT COUNT(*) FROM admin_users WHERE role = 'SUPER_ADMIN'").Scan(&superCount); err == nil && superCount > 0 {
			return
		}
	}

	superUser := strings.TrimSpace(os.Getenv("SUPER_ADMIN_USERNAME"))
	if superUser == "" {
		superUser = "superadmin"
	}
	superPass := os.Getenv("SUPER_ADMIN_PASSWORD")
	if superPass == "" {
		superPass = "123"
	}

	adminUser := strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "123"
	}

	petugasUser := strings.TrimSpace(os.Getenv("PETUGAS_USERNAME"))
	if petugasUser == "" {
		petugasUser = "petugas"
	}
	petugasPass := os.Getenv("PETUGAS_PASSWORD")
	if petugasPass == "" {
		petugasPass = "123"
	}

	seeds := []struct {
		id       string
		username string
		name     string
		email    string
		password string
		role     string
	}{
		{"staff-superadmin-1", superUser, "Kepala Penyelenggara (Super Admin)", superUser + "@symphoniatic.id", superPass, "SUPER_ADMIN"},
		{"staff-admin-1", adminUser, "Administrator Operasional", adminUser + "@symphoniatic.id", adminPass, "ADMIN"},
		{"staff-petugas-1", petugasUser, "Petugas Gate Utama", petugasUser + "@symphoniatic.id", petugasPass, "PETUGAS"},
	}

	for _, s := range seeds {
		hash, err := utils.HashPassword(s.password)
		if err != nil {
			log.Printf("Gagal melakukan hash kata sandi seed staf %s: %v", s.username, err)
			continue
		}
		_, err = DB.Exec(`
			INSERT INTO admin_users (id, username, name, email, password_hash, role, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE, NOW(), NOW())
			ON CONFLICT (username) DO NOTHING
		`, s.id, strings.ToLower(s.username), s.name, strings.ToLower(s.email), hash, s.role)
		if err != nil {
			log.Printf("Gagal melakukan seeding akun staf %s: %v", s.username, err)
		}
	}
}

func ensurePromotionsSeeded() {
	_, err := DB.Exec(`
		INSERT INTO promotions (id, code, description, discount_type, discount_value, min_order_amount, max_discount_amount, is_member_only, is_active, quota, remaining_quota, valid_from, valid_until)
		VALUES
			('promo-symphonia-10', 'SYMPHONIA10', 'Diskon Anggota 10% (maksimal potongan Rp 50.000)', 'PERCENTAGE', 10, 100000, 50000, TRUE, TRUE, 500, 500, NOW() - INTERVAL '1 day', NOW() + INTERVAL '365 days'),
			('promo-harmony-50k', 'HARMONY50K', 'Potongan Anggota Rp 50.000 untuk transaksi minimal Rp 200.000', 'FIXED', 50000, 200000, 50000, TRUE, TRUE, 300, 300, NOW() - INTERVAL '1 day', NOW() + INTERVAL '365 days')
		ON CONFLICT (code) DO NOTHING
	`)
	if err != nil {
		log.Printf("Gagal melakukan seeding promo awal: %v", err)
	}
}

func seedInitialData() {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	log.Println("Melakukan seeding data awal konser & kategori tiket...")

	tx, err := DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	events := []struct {
		ID        string
		Title     string
		Artist    string
		Venue     string
		Date      string
		Time      string
		Category  string
		BadgeCol  string
		Image     string
		AudioURL  string
		Conductor string
		OpenGate  string
		Address   string
		Organizer string
		Subtitle  string
		Rundown   string
		Desc      string
		Cats      []struct {
			ID    string
			Name  string
			Price float64
			Quota int
		}
	}{
		{
			ID:        "evt-1",
			Title:     "Simfoni Mahakarya Beethoven No. 9",
			Artist:    "Orkestra Filharmoni Jakarta & Solois Vokal",
			Venue:     "Aula Simfoni Jakarta, Kemayoran",
			Date:      "15 Agustus 2027",
			Time:      "19:30 WIB",
			Category:  "SIMFONI UTAMA",
			BadgeCol:  "bg-blue-900/80 text-blue-200 border-blue-500/40",
			Image:     "/images/concerts/symphony-no-5.jpg",
			AudioURL:  "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3",
			Conductor: "Maestro Addie MS",
			OpenGate:  "18:00 WIB",
			Address:   "Jl. Benyamin Suaeb, Kemayoran, Jakarta Pusat",
			Organizer: "SymphoniaTic Production",
			Subtitle:  "Pertunjukan Mahakarya Simfoni",
			Rundown:   `[{"time":"18:00 WIB","activity":"Open Gate & Registrasi Tiket"},{"time":"19:30 WIB","activity":"Pertunjukan Utama Orkes Simfoni"},{"time":"21:30 WIB","activity":"Selesai & Curtain Call"}]`,
			Desc:      "Pertunjukan karya legendaris Ode to Joy Beethoven dipimpin oleh Conductor Utama dengan gabungan 80 musisi paduan suara.",
			Cats: []struct {
				ID    string
				Name  string
				Price float64
				Quota int
			}{
				{"cat-1-1", "VIP Pit (Depan Panggung)", 1500000, 15},
				{"cat-1-2", "CAT 1 (Balkon Utama)", 850000, 40},
				{"cat-1-3", "Festival (Lantai Utama)", 450000, 80},
			},
		},
		{
			ID:        "evt-2",
			Title:     "Malam Balet Klasik: Danau Angsa (Swan Lake)",
			Artist:    "Nusantara Ballet Company & Chamber Orchestra",
			Venue:     "Teater Jakarta, Taman Ismail Marzuki",
			Date:      "22 Agustus 2027",
			Time:      "20:00 WIB",
			Category:  "BALET & OPERA",
			BadgeCol:  "bg-purple-900/80 text-purple-200 border-purple-500/40",
			Image:     "/images/concerts/winner-takes-it-all.jpg",
			AudioURL:  "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-2.mp3",
			Conductor: "Maestro Hikotaro Yazaki",
			OpenGate:  "18:30 WIB",
			Address:   "Jl. Cikini Raya No.73, Menteng, Jakarta Pusat",
			Organizer: "SymphoniaTic Production",
			Subtitle:  "Pertunjukan Balet & Opera Romantis",
			Rundown:   `[{"time":"18:30 WIB","activity":"Open Gate & Registrasi Tiket"},{"time":"20:00 WIB","activity":"Pertunjukan Utama Balet & Opera"},{"time":"22:00 WIB","activity":"Selesai & Curtain Call"}]`,
			Desc:      "Pertunjukan balet romantis Tchaikovsky yang memukau dengan alunan musik live dari chamber orchestra bertaraf internasional.",
			Cats: []struct {
				ID    string
				Name  string
				Price float64
				Quota int
			}{
				{"cat-2-1", "VVIP Box (View Terbaik)", 2000000, 10},
				{"cat-2-2", "CAT 1 (Tengah)", 1100000, 35},
				{"cat-2-3", "CAT 2 (Samping)", 600000, 60},
			},
		},
		{
			ID:        "evt-3",
			Title:     "Resital Piano Tunggal: Malam Chopin & Liszt",
			Artist:    "Solois Pianis Muda Internasional",
			Venue:     "Gedung Kesenian Jakarta, Pasar Baru",
			Date:      "05 September 2027",
			Time:      "19:00 WIB",
			Category:  "RESITAL PIANO",
			BadgeCol:  "bg-emerald-900/80 text-emerald-200 border-emerald-500/40",
			Image:     "/images/concerts/piano-recital.jpg",
			AudioURL:  "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-3.mp3",
			Conductor: "-",
			OpenGate:  "17:30 WIB",
			Address:   "Jl. Gedung Kesenian No.1, Pasar Baru, Jakarta Pusat",
			Organizer: "SymphoniaTic Production",
			Subtitle:  "Pertunjukan Piano Solo Virtuoso",
			Rundown:   `[{"time":"17:30 WIB","activity":"Open Gate & Registrasi Tiket"},{"time":"19:00 WIB","activity":"Pertunjukan Utama Piano Solo"},{"time":"21:00 WIB","activity":"Selesai & Curtain Call"}]`,
			Desc:      "Pengalaman akustik akustik mendalam menampilkan Nocturne Chopin dan Rhapsody Liszt di instrumen Steinway & Sons.",
			Cats: []struct {
				ID    string
				Name  string
				Price float64
				Quota int
			}{
				{"cat-3-1", "VIP Diamond", 1250000, 8},
				{"cat-3-2", "CAT 1 Gold", 750000, 25},
				{"cat-3-3", "Student Pass", 300000, 50},
			},
		},
	}

	for _, e := range events {
		parsedDt, _ := utils.ParseConcertDateTime(e.Date, e.Time)
		var eventDtVal interface{}
		if !parsedDt.IsZero() {
			eventDtVal = parsedDt
		}

		_, err := tx.Exec(`
			INSERT INTO events (id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, conductor, open_gate, address, organizer, subtitle, rundown, description, event_date_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		`, e.ID, e.Title, e.Artist, e.Venue, e.Date, e.Time, e.Category, e.BadgeCol, e.Image, e.AudioURL, e.Conductor, e.OpenGate, e.Address, e.Organizer, e.Subtitle, e.Rundown, e.Desc, eventDtVal)
		if err != nil {
			log.Printf("Err insert event: %v", err)
			return
		}

		for _, c := range e.Cats {
			_, err := tx.Exec(`
				INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, c.ID, e.ID, c.Name, c.Price, c.Quota, c.Quota)
			if err != nil {
				log.Printf("Err insert category: %v", err)
				return
			}
		}
	}

	_ = tx.Commit()
	log.Println("Seeding data awal konser selesai")
}

func ensureEventSeatsInitialized() {
	rows, err := DB.Query("SELECT id FROM events")
	if err != nil {
		log.Printf("Gagal memeriksa daftar konser untuk inisialisasi kursi: %v", err)
		return
	}
	var eventIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			eventIDs = append(eventIDs, id)
		}
	}
	rows.Close()

	for _, eventID := range eventIDs {
		var seatCount int
		if err := DB.QueryRow("SELECT COUNT(*) FROM seats WHERE event_id = $1", eventID).Scan(&seatCount); err != nil || seatCount > 0 {
			continue
		}

		catRows, err := DB.Query(`
			SELECT id, event_id, name, price, quota, remaining_quota
			FROM ticket_categories
			WHERE event_id = $1
			ORDER BY price DESC
		`, eventID)
		if err != nil {
			continue
		}
		var cats []models.TicketCategory
		for catRows.Next() {
			var c models.TicketCategory
			if err := catRows.Scan(&c.ID, &c.EventID, &c.Name, &c.Price, &c.Quota, &c.RemainingQuota); err == nil {
				cats = append(cats, c)
			}
		}
		catRows.Close()

		if len(cats) == 0 {
			continue
		}

		seats := utils.GenerateHallSeats(eventID, cats)
		if len(seats) == 0 {
			continue
		}

		tx, err := DB.Begin()
		if err != nil {
			continue
		}

		for _, s := range seats {
			if _, err := tx.Exec(`
				INSERT INTO seats (id, event_id, category_id, seat_code, row_label, seat_number, status)
				VALUES ($1, $2, $3, $4, $5, $6, 'AVAILABLE')
			`, s.ID, s.EventID, s.CategoryID, s.SeatCode, s.RowLabel, s.SeatNumber); err != nil {
				log.Printf("Gagal menyimpan kursi awal untuk konser %s: %v", eventID, err)
				tx.Rollback()
				goto nextEvent
			}
		}

		// Sinkronkan pesanan aktif terbaru maksimal sejumlah tiket terpakai (quota - remaining_quota)
		{
			catByName := make(map[string]models.TicketCategory, len(cats))
			for _, c := range cats {
				catByName[strings.ToLower(strings.TrimSpace(c.Name))] = c
			}
			bookedBudget := make(map[string]int, len(cats))
			for _, c := range cats {
				used := c.Quota - c.RemainingQuota
				if used < 0 {
					used = 0
				}
				bookedBudget[c.ID] = used
			}

			ordRows, err := tx.Query(`
				SELECT id, category_name, quantity
				FROM orders
				WHERE event_id = $1
				  AND UPPER(status) IN ('ISSUED', 'VERIFIED', 'REMINDED', 'CHECKED_IN')
				  AND COALESCE(seat_numbers, '') = ''
				ORDER BY created_at DESC
			`, eventID)
			if err == nil {
				type legacyOrder struct {
					id      string
					catName string
					qty     int
				}
				var legacyOrders []legacyOrder
				for ordRows.Next() {
					var lo legacyOrder
					if err := ordRows.Scan(&lo.id, &lo.catName, &lo.qty); err == nil {
						legacyOrders = append(legacyOrders, lo)
					}
				}
				ordRows.Close()

				for _, lo := range legacyOrders {
					cat, ok := catByName[strings.ToLower(strings.TrimSpace(lo.catName))]
					if !ok || lo.qty <= 0 || bookedBudget[cat.ID] < lo.qty {
						continue
					}
					sRows, err := tx.Query(`
						SELECT id, seat_code
						FROM seats
						WHERE event_id = $1 AND category_id = $2 AND status = 'AVAILABLE'
						ORDER BY row_label ASC, seat_number ASC
						LIMIT $3
					`, eventID, cat.ID, lo.qty)
					if err != nil {
						continue
					}
					var seatIDs, seatCodes []string
					for sRows.Next() {
						var sid, scode string
						if err := sRows.Scan(&sid, &scode); err == nil {
							seatIDs = append(seatIDs, sid)
							seatCodes = append(seatCodes, scode)
						}
					}
					sRows.Close()

					if len(seatIDs) == lo.qty {
						for _, sid := range seatIDs {
							_, _ = tx.Exec("UPDATE seats SET status = 'BOOKED', order_id = $1 WHERE id = $2", lo.id, sid)
						}
						_, _ = tx.Exec("UPDATE orders SET seat_numbers = $1 WHERE id = $2", strings.Join(seatCodes, ", "), lo.id)
						bookedBudget[cat.ID] -= lo.qty
					}
				}
			}
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Gagal commit inisialisasi kursi konser %s: %v", eventID, err)
		} else {
			log.Printf("Inisialisasi %d nomor kursi untuk konser %s selesai", len(seats), eventID)
		}

	nextEvent:
	}
}
