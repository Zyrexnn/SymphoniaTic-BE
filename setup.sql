-- ====================================================================
-- SYMPHONIATIC DATABASE SCHEMA & INITIAL DATA SEED (POSTGRESQL NATIVE)
-- ====================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. SCHEMA CREATION
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
    conductor VARCHAR(255) DEFAULT '',
    open_gate VARCHAR(100) DEFAULT '',
    address TEXT DEFAULT '',
    organizer VARCHAR(255) DEFAULT '',
    subtitle TEXT DEFAULT '',
    rundown JSONB DEFAULT '[]'::jsonb,
    description TEXT NOT NULL,
    is_closed BOOLEAN DEFAULT FALSE,
    event_date_time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    sales_end_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
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
    user_id VARCHAR(64) DEFAULT '',
    event_id VARCHAR(64) NOT NULL REFERENCES events(id),
    event_title VARCHAR(255) NOT NULL,
    artist VARCHAR(255) NOT NULL,
    venue VARCHAR(255) NOT NULL,
    date VARCHAR(100) NOT NULL,
    category_name VARCHAR(100) NOT NULL,
    quantity INT NOT NULL,
    total_price NUMERIC(12, 2) NOT NULL,
    promo_code VARCHAR(50) DEFAULT '',
    discount_amount NUMERIC(12, 2) DEFAULT 0,
    user_name VARCHAR(255) NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    qr_code VARCHAR(255) NOT NULL,
    seat_numbers VARCHAR(255) DEFAULT '',
    attendee_names TEXT DEFAULT '',
    checked_in_seats VARCHAR(255) DEFAULT '',
    seat_checkins TEXT DEFAULT '',
    checked_in_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ISSUED',
    payment_method VARCHAR(50) DEFAULT 'SANDBOX_PAYMENT',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

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

CREATE INDEX IF NOT EXISTS idx_events_category ON events(category);
CREATE INDEX IF NOT EXISTS idx_ticket_categories_event_id ON ticket_categories(event_id);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_order_code ON orders(order_code);
CREATE INDEX IF NOT EXISTS idx_orders_user_email ON orders(user_email);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at_status ON orders(created_at, status);
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

CREATE INDEX IF NOT EXISTS idx_refund_requests_order_code ON refund_requests(order_code);
CREATE INDEX IF NOT EXISTS idx_refund_requests_status ON refund_requests(status);

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

INSERT INTO promotions (id, code, description, discount_type, discount_value, min_order_amount, max_discount_amount, is_member_only, is_active, quota, remaining_quota, valid_from, valid_until)
VALUES
    ('promo-symphonia-10', 'SYMPHONIA10', 'Diskon Anggota 10% (maksimal potongan Rp 50.000)', 'PERCENTAGE', 10, 100000, 50000, TRUE, TRUE, 500, 500, NOW() - INTERVAL '1 day', NOW() + INTERVAL '365 days'),
    ('promo-harmony-50k', 'HARMONY50K', 'Potongan Anggota Rp 50.000 untuk transaksi minimal Rp 200.000', 'FIXED', 50000, 200000, 50000, TRUE, TRUE, 300, 300, NOW() - INTERVAL '1 day', NOW() + INTERVAL '365 days')
ON CONFLICT (code) DO NOTHING;

-- 3. RESET & DATA SEEDING
TRUNCATE TABLE orders, ticket_categories, events, refund_requests, auth_otps CASCADE;

INSERT INTO events (id, title, subtitle, artist, conductor, venue, address, date, time, open_gate, category, category_badge_color, image, audio_url, organizer, description, rundown, event_date_time)
VALUES 
(
    'evt-1',
    'Symphony No. 5 in C minor',
    'Opus 67 — I. Allegro con brio Masterpiece',
    'Royal Philharmonic Orchestra & Jakarta Choral Society',
    'Maestro Alexander Vance',
    'Aula Simfonia Jakarta',
    'Jl. Industri Blok B14 No.1, Kemayoran, Jakarta Pusat 10720',
    'Sabtu, 18 April 2027',
    '19:30 WIB',
    '18:00 WIB',
    'SIMFONI',
    'bg-amber-500/20 text-amber-300 border-amber-500/40',
    '/images/concerts/symphony-no-5.jpg',
    '/audio/Ludwig van Beethoven - Symphony n.5 in C minor, Op.67, I.Allegro con brio.mp3',
    'Royal Philharmonic Foundation & SymphoniaTic Events',
    'Mahakarya simfoni Ludwig van Beethoven yang sangat terkenal. Menampilkan irama 4 ketukan ikonik Allegro con brio dengan kolaborasi 90 musisi orkestra simfoni profesional.',
    '[
      {"time": "18:00 WIB", "activity": "Pemeriksaan E-Ticket & Registrasi Open Gate"},
      {"time": "19:00 WIB", "activity": "Pintu Main Hall Dibuka & Pre-Concert Presentation"},
      {"time": "19:30 WIB", "activity": "Babak I: Movement I (Allegro con brio)"},
      {"time": "20:30 WIB", "activity": "Istirahat / Intermission (20 Menit)"},
      {"time": "20:50 WIB", "activity": "Babak II: Movement II & III Finale"},
      {"time": "21:45 WIB", "activity": "Selesai & Sesi Foto Konduktor"}
    ]'::jsonb,
    '2027-04-18 19:30:00+07'::timestamptz
),
(
    'evt-2',
    'Viva La Vida (Orchestra Festa)',
    'Coldplay & Oasis Band Music Orchestra Celebration',
    'Vivaldi & Band Orchestra Ensemble',
    'Violinis Utama Iskandar Widjaja',
    'TIM Concert Hall (Taman Ismail Marzuki)',
    'Jl. Cikini Raya No.73, Menteng, Jakarta Pusat',
    'Minggu, 19 April 2027',
    '20:00 WIB',
    '18:30 WIB',
    'KAMAR MUSIK',
    'bg-emerald-500/20 text-emerald-300 border-emerald-500/40',
    '/images/concerts/viva-la-vida.jpg',
    '/audio/coldplay - Viva La Vida I COLDPLAY & OASIS AND BAND MUSIC ORCHESTRA FESTA.mp3',
    'Jakarta Chamber Society & Modern Orchestra',
    'Reinterpretasi megah lagu hit Viva La Vida yang dibawakan oleh gabungan tim orkestra simfoni dan ansambel string modern.',
    '[
      {"time": "18:30 WIB", "activity": "Open Gate & Registrasi Ulang E-Ticket"},
      {"time": "19:30 WIB", "activity": "Pengenalan Karya & Pengantar Neoklasik"},
      {"time": "20:00 WIB", "activity": "Pertunjukan Utama Viva La Vida Orchestra"},
      {"time": "21:00 WIB", "activity": "Pertunjukan Sesi II Band Symphony"},
      {"time": "22:00 WIB", "activity": "Penutupan & Sesi Tanya Jawab Musik"}
    ]'::jsonb,
    '2027-04-19 20:00:00+07'::timestamptz
),
(
    'evt-3',
    'The Winner Takes It All (Epic Orchestra)',
    'Pertunjukan Balet & Orkestra Epik Mahakarya ABBA',
    'Grand Opera Orchestra & Jakarta Ballet Company',
    'Maestro David Chen',
    'JIExpo Symphony Hall',
    'Gedung Pusat Niaga Pekan Raya Jakarta, Kemayoran, Jakarta Pusat',
    'Jumat, 24 April 2027',
    '19:00 WIB',
    '17:30 WIB',
    'BALET & OPERA',
    'bg-purple-500/20 text-purple-300 border-purple-500/40',
    '/images/concerts/winner-takes-it-all.jpg',
    '/audio/ABBA - The Winner Takes It All  Epic Orchestra (2020).mp3',
    'Indonesian Classical Ballet Theatre',
    'Aransemen epik orkestra dari lagu legendaris ABBA ''The Winner Takes It All'' diiringi koreografi tarian balet simfoni kolosal.',
    '[
      {"time": "17:30 WIB", "activity": "Open Gate & Booth Merchandise Balet"},
      {"time": "19:00 WIB", "activity": "Babak I: Epic Orchestra Suite Act 1"},
      {"time": "20:15 WIB", "activity": "Istirahat (15 Menit)"},
      {"time": "20:30 WIB", "activity": "Babak II: Winner Takes It All Highlights"},
      {"time": "21:30 WIB", "activity": "Penutupan & Curtain Call"}
    ]'::jsonb,
    '2027-04-24 19:00:00+07'::timestamptz
),
(
    'evt-4',
    'Laskar Pelangi (TRUST Symphony)',
    'Konser Simfoni Mahakarya Kebangsaan Indonesia',
    'TRUST (Trinity Youth Symphony Orchestra)',
    'Dr. Nathania Karina',
    'Aula Simfonia Jakarta',
    'Jl. Industri Blok B14 No.1, Kemayoran, Jakarta Pusat',
    'Sabtu, 2 Mei 2027',
    '19:00 WIB',
    '17:30 WIB',
    'NUSANTARA SYMPHONY',
    'bg-blue-500/20 text-blue-300 border-blue-500/40',
    '/images/concerts/laskar-pelangi.jpg',
    '/audio/Laskar Pelangi  TRUST (Trinity Youth Symphony Orchestra).mp3',
    'TRUST Orchestra & SymphoniaTic Events',
    'Aransemen orkestra simfoni memukau dari lagu kebangsaan legendaris Laskar Pelangi karya Nidji, dibawakan secara megah oleh Trinity Youth Symphony Orchestra.',
    '[
      {"time": "17:30 WIB", "activity": "Open Gate & Booth Merchandise Nusantara"},
      {"time": "19:00 WIB", "activity": "Babak I: Simfoni Pemuda & Lagu Nusantara"},
      {"time": "20:15 WIB", "activity": "Istirahat (15 Menit)"},
      {"time": "20:30 WIB", "activity": "Babak II: Pertunjukan Utama Laskar Pelangi Symphony"},
      {"time": "21:45 WIB", "activity": "Selesai & Sesi Foto Musisi"}
    ]'::jsonb,
    '2027-05-02 19:00:00+07'::timestamptz
);

-- Seed Categories
INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota)
VALUES
('c1-vip', 'evt-1', 'VIP Orchestral Pit', 750000.00, 14, 14),
('c1-cat1', 'evt-1', 'CAT 1 Grand Tier', 450000.00, 24, 24),
('c1-fest', 'evt-1', 'Festival Stalls', 300000.00, 50, 50),

('c2-vip', 'evt-2', 'VIP Front Row', 850000.00, 5, 5),
('c2-cat1', 'evt-2', 'CAT 1 Main Hall', 500000.00, 120, 120),

('c3-vip', 'evt-3', 'Royal Box VIP', 600000.00, 8, 8),
('c3-cat1', 'evt-3', 'CAT 1 Balcony', 350000.00, 12, 12),

('c4-vip', 'evt-4', 'VIP Nusantara Pit', 650000.00, 10, 10),
('c4-cat1', 'evt-4', 'CAT 1 Main Tier', 400000.00, 30, 30);
