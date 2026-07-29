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

-- 2. INDEXES
CREATE INDEX IF NOT EXISTS idx_events_category ON events(category);
CREATE INDEX IF NOT EXISTS idx_ticket_categories_event_id ON ticket_categories(event_id);
CREATE INDEX IF NOT EXISTS idx_orders_order_code ON orders(order_code);
CREATE INDEX IF NOT EXISTS idx_orders_user_email ON orders(user_email);

-- 3. RESET & DATA SEEDING
TRUNCATE TABLE orders, ticket_categories, events CASCADE;

INSERT INTO events (id, title, subtitle, artist, conductor, venue, address, date, time, open_gate, category, category_badge_color, image, audio_url, organizer, description, rundown)
VALUES 
(
    'evt-1',
    'Symphony No. 5 in C minor',
    'Opus 67 — I. Allegro con brio Masterpiece',
    'Royal Philharmonic Orchestra & Jakarta Choral Society',
    'Maestro Alexander Vance',
    'Aula Simfonia Jakarta',
    'Jl. Industri Blok B14 No.1, Kemayoran, Jakarta Pusat 10720',
    'Sabtu, 18 April 2026',
    '19:30 WIB',
    '18:00 WIB',
    'SIMFONI',
    'bg-amber-500/20 text-amber-300 border-amber-500/40',
    'https://images.unsplash.com/photo-1465847899084-d164df4dedc6?auto=format&fit=crop&w=1200&q=80',
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
    ]'::jsonb
),
(
    'evt-2',
    'Viva La Vida (Orchestra Festa)',
    'Coldplay & Oasis Band Music Orchestra Celebration',
    'Vivaldi & Band Orchestra Ensemble',
    'Violinis Utama Iskandar Widjaja',
    'TIM Concert Hall (Taman Ismail Marzuki)',
    'Jl. Cikini Raya No.73, Menteng, Jakarta Pusat',
    'Minggu, 19 April 2026',
    '20:00 WIB',
    '18:30 WIB',
    'KAMAR MUSIK',
    'bg-emerald-500/20 text-emerald-300 border-emerald-500/40',
    'https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?auto=format&fit=crop&w=1200&q=80',
    '/audio/coldplay - Viva La Vida I COLDPLAY & OASIS AND BAND MUSIC ORCHESTRA FESTA.mp3',
    'Jakarta Chamber Society & Modern Orchestra',
    'Reinterpretasi megah lagu hit Viva La Vida yang dibawakan oleh gabungan tim orkestra simfoni dan ansambel string modern.',
    '[
      {"time": "18:30 WIB", "activity": "Open Gate & Registrasi Ulang E-Ticket"},
      {"time": "19:30 WIB", "activity": "Pengenalan Karya & Pengantar Neoklasik"},
      {"time": "20:00 WIB", "activity": "Pertunjukan Utama Viva La Vida Orchestra"},
      {"time": "21:00 WIB", "activity": "Pertunjukan Sesi II Band Symphony"},
      {"time": "22:00 WIB", "activity": "Penutupan & Sesi Tanya Jawab Musik"}
    ]'::jsonb
),
(
    'evt-3',
    'The Winner Takes It All (Epic Orchestra)',
    'Pertunjukan Balet & Orkestra Epik Mahakarya ABBA',
    'Grand Opera Orchestra & Jakarta Ballet Company',
    'Maestro David Chen',
    'JIExpo Symphony Hall',
    'Gedung Pusat Niaga Pekan Raya Jakarta, Kemayoran, Jakarta Pusat',
    'Jumat, 24 April 2026',
    '19:00 WIB',
    '17:30 WIB',
    'BALET & OPERA',
    'bg-purple-500/20 text-purple-300 border-purple-500/40',
    'https://images.unsplash.com/photo-1514525253161-7a46d19cd819?auto=format&fit=crop&w=1200&q=80',
    '/audio/ABBA - The Winner Takes It All  Epic Orchestra (2020).mp3',
    'Indonesian Classical Ballet Theatre',
    'Aransemen epik orkestra dari lagu legendaris ABBA ''The Winner Takes It All'' diiringi koreografi tarian balet simfoni kolosal.',
    '[
      {"time": "17:30 WIB", "activity": "Open Gate & Booth Merchandise Balet"},
      {"time": "19:00 WIB", "activity": "Babak I: Epic Orchestra Suite Act 1"},
      {"time": "20:15 WIB", "activity": "Istirahat (15 Menit)"},
      {"time": "20:30 WIB", "activity": "Babak II: Winner Takes It All Highlights"},
      {"time": "21:30 WIB", "activity": "Penutupan & Curtain Call"}
    ]'::jsonb
),
(
    'evt-4',
    'Laskar Pelangi (TRUST Symphony)',
    'Konser Simfoni Mahakarya Kebangsaan Indonesia',
    'TRUST (Trinity Youth Symphony Orchestra)',
    'Dr. Nathania Karina',
    'Aula Simfonia Jakarta',
    'Jl. Industri Blok B14 No.1, Kemayoran, Jakarta Pusat',
    'Sabtu, 2 Mei 2026',
    '19:00 WIB',
    '17:30 WIB',
    'NUSANTARA SYMPHONY',
    'bg-blue-500/20 text-blue-300 border-blue-500/40',
    'https://images.unsplash.com/photo-1507676184212-d03ab07a01bf?auto=format&fit=crop&w=1200&q=80',
    '/audio/Laskar Pelangi  TRUST (Trinity Youth Symphony Orchestra).mp3',
    'TRUST Orchestra & SymphoniaTic Events',
    'Aransemen orkestra simfoni memukau dari lagu kebangsaan legendaris Laskar Pelangi karya Nidji, dibawakan secara megah oleh Trinity Youth Symphony Orchestra.',
    '[
      {"time": "17:30 WIB", "activity": "Open Gate & Booth Merchandise Nusantara"},
      {"time": "19:00 WIB", "activity": "Babak I: Simfoni Pemuda & Lagu Nusantara"},
      {"time": "20:15 WIB", "activity": "Istirahat (15 Menit)"},
      {"time": "20:30 WIB", "activity": "Babak II: Pertunjukan Utama Laskar Pelangi Symphony"},
      {"time": "21:45 WIB", "activity": "Selesai & Sesi Foto Musisi"}
    ]'::jsonb
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
