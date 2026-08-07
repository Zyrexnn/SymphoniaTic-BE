package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func ConnectDB() {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi PostgreSQL: %v", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute)

	if err = DB.Ping(); err != nil {
		log.Fatalf("Gagal ping PostgreSQL: %v", err)
	}

	log.Println("✅ Berhasil terhubung ke database PostgreSQL!")

	initSchema()
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
		status VARCHAR(50) NOT NULL DEFAULT 'VERIFIED',
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

	ALTER TABLE orders ADD COLUMN IF NOT EXISTS user_id VARCHAR(64) DEFAULT '';
	CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
	CREATE INDEX IF NOT EXISTS idx_orders_event_id ON orders(event_id);
	CREATE INDEX IF NOT EXISTS idx_orders_user_email_lower ON orders(LOWER(user_email));
	CREATE INDEX IF NOT EXISTS idx_ticket_categories_event_id ON ticket_categories(event_id);

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
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

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
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS auth_otps (
		id VARCHAR(64) PRIMARY KEY,
		email VARCHAR(255) NOT NULL,
		otp_code VARCHAR(10) NOT NULL,
		purpose VARCHAR(50) NOT NULL,
		attempts INT DEFAULT 0,
		is_used BOOLEAN DEFAULT FALSE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_auth_otps_email_purpose ON auth_otps(email, purpose);
	`

	_, err := DB.Exec(schemaQuery)
	if err != nil {
		log.Fatalf("Gagal inisialisasi skema tabel PostgreSQL: %v", err)
	}

	log.Println("✅ Skema database PostgreSQL (events, ticket_categories, orders, users, auth_otps) siap!")
	seedInitialData()
}

func seedInitialData() {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	log.Println("🌱 Melakukan seeding data awal konser & kategori tiket...")

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
			Date:      "15 Agustus 2026",
			Time:      "19:30 WIB",
			Category:  "SIMFONI UTAMA",
			BadgeCol:  "bg-blue-900/80 text-blue-200 border-blue-500/40",
			Image:     "https://images.unsplash.com/photo-1465847899084-d164df4dedc6?q=80&w=1000&auto=format&fit=crop",
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
			Date:      "22 Agustus 2026",
			Time:      "20:00 WIB",
			Category:  "BALET & OPERA",
			BadgeCol:  "bg-purple-900/80 text-purple-200 border-purple-500/40",
			Image:     "https://images.unsplash.com/photo-1516450360452-9312f5e86fc7?q=80&w=1000&auto=format&fit=crop",
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
			Date:      "05 September 2026",
			Time:      "19:00 WIB",
			Category:  "RESITAL PIANO",
			BadgeCol:  "bg-emerald-900/80 text-emerald-200 border-emerald-500/40",
			Image:     "https://images.unsplash.com/photo-1520523839897-bd0b52f945a0?q=80&w=1000&auto=format&fit=crop",
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
		_, err := tx.Exec(`
			INSERT INTO events (id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, conductor, open_gate, address, organizer, subtitle, rundown, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		`, e.ID, e.Title, e.Artist, e.Venue, e.Date, e.Time, e.Category, e.BadgeCol, e.Image, e.AudioURL, e.Conductor, e.OpenGate, e.Address, e.Organizer, e.Subtitle, e.Rundown, e.Desc)
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
	log.Println("✅ Seeding data awal konser selesai!")
}
