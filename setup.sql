-- ====================================================================
-- SYMPHONIATIC DATABASE SCHEMA & INITIAL DATA SEED (POSTGRESQL NATIVE)
-- ====================================================================
-- Dokumentasi: File SQL ini digunakan untuk mengatur struktur tabel
-- dan memasukkan data sampel awal agar siap dipakai oleh tim developer/QA.

-- 1. EXTENSIONS
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. DROP TABLES IF EXISTS (CLEAN RESET OPTIONAL)
-- DROP TABLE IF EXISTS orders CASCADE;
-- DROP TABLE IF EXISTS ticket_categories CASCADE;
-- DROP TABLE IF EXISTS events CASCADE;

-- 3. SCHEMA CREATION

-- 3.1 Tabel Events (Data Konser & Pertunjukan)
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

-- 3.2 Tabel Ticket Categories (Kategori Kursi & Alokasi Kuota)
CREATE TABLE IF NOT EXISTS ticket_categories (
    id VARCHAR(64) PRIMARY KEY,
    event_id VARCHAR(64) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    price NUMERIC(12, 2) NOT NULL,
    quota INT NOT NULL,
    remaining_quota INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3.3 Tabel Orders (Transaksi Pemesanan Tiket Guest Checkout)
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

-- 4. INDEXES FOR HIGH PERFORMANCE QUERY
CREATE INDEX IF NOT EXISTS idx_events_category ON events(category);
CREATE INDEX IF NOT EXISTS idx_ticket_categories_event_id ON ticket_categories(event_id);
CREATE INDEX IF NOT EXISTS idx_orders_order_code ON orders(order_code);
CREATE INDEX IF NOT EXISTS idx_orders_user_email ON orders(user_email);

-- 5. INITIAL DATA SEEDING (SAMPLE CONCERTS & SEAT CATEGORIES)

-- 5.1 Seed Data Events
INSERT INTO events (id, title, artist, venue, date, time, category, category_badge_color, image, audio_url, description)
VALUES 
(
    'evt-1',
    'Simfoni Mahakarya Beethoven No. 9',
    'Orkestra Filharmoni Jakarta & Solois Vokal',
    'Aula Simfoni Jakarta, Kemayoran',
    '15 Agustus 2026',
    '19:30 WIB',
    'SIMFONI UTAMA',
    'bg-blue-900/80 text-blue-200 border-blue-500/40',
    'https://images.unsplash.com/photo-1465847899084-d164df4dedc6?q=80&w=1000&auto=format&fit=crop',
    'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3',
    'Pertunjukan karya legendaris Ode to Joy Beethoven dipimpin oleh Conductor Utama dengan gabungan 80 musisi paduan suara.'
),
(
    'evt-2',
    'Malam Balet Klasik: Danau Angsa (Swan Lake)',
    'Nusantara Ballet Company & Chamber Orchestra',
    'Teater Jakarta, Taman Ismail Marzuki',
    '22 Agustus 2026',
    '20:00 WIB',
    'BALET & OPERA',
    'bg-purple-900/80 text-purple-200 border-purple-500/40',
    'https://images.unsplash.com/photo-1516450360452-9312f5e86fc7?q=80&w=1000&auto=format&fit=crop',
    'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-2.mp3',
    'Pertunjukan balet romantis Tchaikovsky yang memukau dengan alunan musik live dari chamber orchestra bertaraf internasional.'
),
(
    'evt-3',
    'Resital Piano Tunggal: Malam Chopin & Liszt',
    'Solois Pianis Muda Internasional',
    'Gedung Kesenian Jakarta, Pasar Baru',
    '05 September 2026',
    '19:00 WIB',
    'RESITAL PIANO',
    'bg-emerald-900/80 text-emerald-200 border-emerald-500/40',
    'https://images.unsplash.com/photo-1520523839897-bd0b52f945a0?q=80&w=1000&auto=format&fit=crop',
    'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-3.mp3',
    'Pengalaman akustik akustik mendalam menampilkan Nocturne Chopin dan Rhapsody Liszt di instrumen Steinway & Sons.'
)
ON CONFLICT (id) DO NOTHING;

-- 5.2 Seed Data Ticket Categories
INSERT INTO ticket_categories (id, event_id, name, price, quota, remaining_quota)
VALUES
-- Categories Event 1
('cat-1-1', 'evt-1', 'VIP Pit (Depan Panggung)', 1500000.00, 15, 15),
('cat-1-2', 'evt-1', 'CAT 1 (Balkon Utama)', 850000.00, 40, 40),
('cat-1-3', 'evt-1', 'Festival (Lantai Utama)', 450000.00, 80, 80),

-- Categories Event 2
('cat-2-1', 'evt-2', 'VVIP Box (View Terbaik)', 2000000.00, 10, 10),
('cat-2-2', 'evt-2', 'CAT 1 (Tengah)', 1100000.00, 35, 35),
('cat-2-3', 'evt-2', 'CAT 2 (Samping)', 600000.00, 60, 60),

-- Categories Event 3
('cat-3-1', 'evt-3', 'VIP Diamond', 1250000.00, 8, 8),
('cat-3-2', 'evt-3', 'CAT 1 Gold', 750000.00, 25, 25),
('cat-3-3', 'evt-3', 'Student Pass', 300000.00, 50, 50)
ON CONFLICT (id) DO NOTHING;

-- 5.3 Sample Seed Data Orders (Pembelian Demo)
INSERT INTO orders (id, order_code, event_id, event_title, artist, venue, date, category_name, quantity, total_price, user_name, user_email, qr_code, status, payment_method)
VALUES
(
    'ord-sample-1',
    'SYM-123456',
    'evt-1',
    'Simfoni Mahakarya Beethoven No. 9',
    'Orkestra Filharmoni Jakarta & Solois Vokal',
    'Aula Simfoni Jakarta, Kemayoran',
    '15 Agustus 2026 @ 19:30 WIB',
    'VIP Pit (Depan Panggung)',
    2,
    3000000.00,
    'Budi Santoso',
    'budi@example.com',
    'QR-SYM-123456',
    'VERIFIED',
    'SANDBOX_PAYMENT'
)
ON CONFLICT (id) DO NOTHING;
