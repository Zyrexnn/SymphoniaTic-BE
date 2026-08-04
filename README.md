# 🎼 SymphoniaTic Backend REST API

Dokumentasi resmi **SymphoniaTic Backend REST API** yang komprehensif untuk seluruh tim (**Front-End**, **Back-End**, **QA**, & **DevOps**). Dokumentasi ini mencakup seluruh daftar endpoint (Public, User Auth, User Account, Order & Ticket Lookup, Refund Publik, serta Admin Panel Management), format request/response JSON lengkap, alur Autentikasi OTP & Password, sistem transaksi Checkout, hingga pengelolaan Admin Dashboard & Mailpit.

---

## ⚡ Ringkasan Server & Setup Lokal

| Environment | URL / Alamat | Keterangan |
| :--- | :--- | :--- |
| **API Base URL** | `http://localhost:8082/api/v1` | Prefix utama seluruh endpoint API |
| **Server Engine** | Go (Golang) + Fiber v2 | High-performance Web Framework |
| **Database** | PostgreSQL | Port default: `5432` |
| **Mailpit Dashboard** | `http://localhost:8025` | Interface browser untuk melihat OTP, E-Ticket, & Notifikasi Email |
| **Mailpit SMTP** | `localhost:1025` | Port SMTP lokal untuk pengujian email |
| **Auth Header (User)** | `Authorization: Bearer <user_jwt_token>` | Header wajib untuk endpoint akun & profil user |
| **Auth Header (Admin)** | `Authorization: Bearer <admin_token>` | Token sesi admin untuk panel manajemen |

---

## 🚀 Panduan Memulai Backend & Mailpit

### 1. Prasyarat Environment Variables (`.env`)
Buat file `.env` di dalam folder `be/` dengan konfigurasi berikut:
```env
PORT=8082
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=symphoniatic
JWT_SECRET=super_secret_jwt_key_symphoniatic_2026
ADMIN_USERNAME=admin
ADMIN_PASSWORD=123
SMTP_HOST=localhost
SMTP_PORT=1025
API_BASE_URL=http://localhost:8082/api/v1
```

### 2. Jalankan PostgreSQL & Mailpit via Docker
```bash
# Jalankan PostgreSQL Database
docker run -d --name symphoniatic-db -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=symphoniatic postgres:alpine

# Jalankan Mailpit Service (Email Testing Tool)
docker run -d --name mailpit -p 8025:8025 -p 1025:1025 axllent/mailpit
```

### 3. Migrasi & Seed Database SQL
Eksekusi file script `setup.sql` ke dalam PostgreSQL untuk membuat skema tabel dan data awal (event konser & kategori tiket):
```bash
psql -h localhost -U postgres -d symphoniatic -f setup.sql
```

### 4. Jalankan Server Go Backend
```bash
cd be
go run ./cmd/server
```
*Server API aktif di `http://localhost:8082`.*

---

## 📋 Standard API Response Envelope

Seluruh endpoint pada SymphoniaTic REST API mengembalikan format JSON yang konsisten menggunakan envelope `APIResponse`:

### Request Berhasil (200 OK / 201 Created)
```json
{
  "success": true,
  "message": "Pesan deskripsi aksi yang berhasil dilakukan.",
  "data": { ... }
}
```

### Request Gagal / Error (400 / 401 / 404 / 429 / 500)
```json
{
  "success": false,
  "message": "Pesan kesalahan yang mudah dipahami.",
  "error": "Detail error teknis (opsional/internal debug)"
}
```

---

## 🗺️ Ringkasan Tabel Endpoint (Quick Reference)

### 📢 1. Event & Konser (Public)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/events` | Ambil semua daftar event konser, kategori tiket, & rundown | Public |
| `GET` | `/api/v1/events/:id` | Ambil detail 1 event konser berdasarkan ID | Public |

### 🔑 2. Autentikasi Pengguna (`/api/v1/auth`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/auth/register/request-otp` | Request OTP untuk registrasi akun baru | Public |
| `POST` | `/api/v1/auth/register/verify-otp` | Verifikasi OTP registrasi & setel kata sandi | Public |
| `POST` | `/api/v1/auth/login/password` | Login menggunakan Email & Password | Public |
| `POST` | `/api/v1/auth/login/request-otp` | Request OTP untuk login tanpa password | Public |
| `POST` | `/api/v1/auth/login/verify-otp` | Verifikasi OTP login & dapatkan JWT Token | Public |
| `POST` | `/api/v1/auth/forgot-password/request-otp` | Request OTP reset kata sandi | Public |
| `POST` | `/api/v1/auth/forgot-password/verify-otp` | Verifikasi OTP reset kata sandi (Mendapatkan Reset Token) | Public |
| `POST` | `/api/v1/auth/forgot-password/reset` | Setel ulang kata sandi baru via Reset Token | Public |
| `GET` | `/api/v1/auth/me` | Ambil profil pengguna yang sedang login | `Bearer Token` |

### 👤 3. Manajemen Akun & Profil (`/api/v1/user`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `PUT` | `/api/v1/user/profile` | Update Nama Lengkap & Nomor HP pengguna | `Bearer Token` |
| `POST` | `/api/v1/user/change-password` | Ubah kata sandi dari dalam akun pengguna | `Bearer Token` |
| `GET` | `/api/v1/user/dashboard-summary` | Ambil ringkasan statistik tiket, konser, & refund user | `Bearer Token` |
| `GET` | `/api/v1/user/orders` | Ambil seluruh riwayat pesanan tiket milik pengguna (Filter: `?status=`) | `Bearer Token` |
| `GET` | `/api/v1/user/orders/:orderCode` | Ambil detail 1 pesanan tiket milik pengguna | `Bearer Token` |
| `GET` | `/api/v1/user/refunds` | Monitoring riwayat & status pengajuan refund pengguna | `Bearer Token` |

### 🎟️ 4. Pemesanan Tiket & Lookup (`/api/v1/orders`, `/api/v1/tickets`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/orders` | Transaksi pemesanan tiket (Autofill User jika ada Token, Guest jika tidak) | Public / User |
| `GET` | `/api/v1/tickets/lookup` | Public lookup tiket berdasarkan kode order (`?code=SYM-xxx`) | Public |

### 🔄 5. Pengajuan Refund Publik (`/api/v1/refunds`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/refunds/request-otp` | Request OTP verifikasi permohonan refund | Public |
| `POST` | `/api/v1/refunds/submit` | Kirim pengajuan refund (OTP + Rekening Bank) | Public |
| `POST` | `/api/v1/refunds/status` | Cek status permohonan refund via Kode Order & Email | Public |

### 🛠️ 6. Admin Panel Management (`/api/v1/admin`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/admin/login` | Login Admin Panel | Public |
| `GET` | `/api/v1/admin/dashboard` | Metric analytics real-time (Revenue, Tickets, Charts) | Admin |
| `POST` | `/api/v1/admin/upload` | Upload gambar poster event (Multipart Form Data) | Admin |
| `POST` | `/api/v1/admin/events` | Buat event konser baru beserta kategori tiket | Admin |
| `PUT` | `/api/v1/admin/events/:id` | Update detail event konser | Admin |
| `PATCH` | `/api/v1/admin/events/:id/toggle-close` | Tutup / Buka kembali penjualan tiket konser | Admin |
| `DELETE` | `/api/v1/admin/events/:id` | Hapus event konser beserta seluruh kategori terkait | Admin |
| `POST` | `/api/v1/admin/events/:id/categories` | Tambah kategori tiket baru ke event tertentu | Admin |
| `PUT` | `/api/v1/admin/categories/:id` | Update kategori tiket (Harga, Nama, & Penyesuaian Kuota) | Admin |
| `DELETE` | `/api/v1/admin/categories/:id` | Hapus kategori tiket | Admin |
| `GET` | `/api/v1/admin/orders` | Daftar semua transaksi (Filter: `?search=` & `?status=`) | Admin |
| `PATCH` | `/api/v1/admin/orders/:id/status` | Update status transaksi (`CHECKED_IN`, `REMINDED`, dll) | Admin |
| `GET` | `/api/v1/admin/refunds` | Daftar seluruh pengajuan refund dari pembeli | Admin |
| `PATCH` | `/api/v1/admin/refunds/:id/status` | Setujui (`APPROVED`) / Tolak (`REJECTED`) refund + Auto-Restock Kuota | Admin |

---

## 📢 Modul 1: Event & Konser (Public)

### 1. Ambil Daftar Semua Event Konser
- **URL**: `GET /api/v1/events`
- **Auth Required**: Tidak
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "Berhasil mengambil data konser",
  "data": [
    {
      "id": "evt-1",
      "title": "Symphony No. 5 in C minor",
      "subtitle": "Opus 67 — I. Allegro con brio Masterpiece",
      "artist": "Royal Philharmonic Orchestra & Jakarta Choral Society",
      "conductor": "Maestro Alexander Vance",
      "venue": "Aula Simfonia Jakarta",
      "address": "Jl. Industri Blok B14 No.1, Kemayoran, Jakarta Pusat 10720",
      "date": "Sabtu, 18 April 2026",
      "time": "19:30 WIB",
      "openGate": "18:00 WIB",
      "category": "SIMFONI",
      "categoryBadgeColor": "bg-amber-500/20 text-amber-300 border-amber-500/40",
      "image": "https://images.unsplash.com/photo-1465847899084-d164df4dedc6",
      "audioUrl": "/audio/Ludwig van Beethoven - Symphony n.5.mp3",
      "organizer": "Royal Philharmonic Foundation & SymphoniaTic Events",
      "description": "Mahakarya simfoni Ludwig van Beethoven...",
      "isClosed": false,
      "rundown": [
        { "time": "18:00 WIB", "activity": "Pemeriksaan E-Ticket & Registrasi Open Gate" },
        { "time": "19:30 WIB", "activity": "Pertunjukan Utama dimulai" }
      ],
      "categories": [
        {
          "id": "c1-vip",
          "eventId": "evt-1",
          "name": "VIP Orchestral Pit",
          "price": 750000,
          "quota": 14,
          "remainingQuota": 14,
          "createdAt": "2026-08-04T07:00:00Z"
        }
      ]
    }
  ]
}
```

### 2. Detail 1 Event Konser
- **URL**: `GET /api/v1/events/:id` (Contoh: `/api/v1/events/evt-1`)
- **Auth Required**: Tidak
- **Response Success (200 OK)**: Mengembalikan objek tunggal `EventItem`.
- **Response Error (404 Not Found)**:
```json
{
  "success": false,
  "message": "Konser tidak ditemukan"
}
```

---

## 🔑 Modul 2: Autentikasi Pengguna (`/api/v1/auth`)

### 1. Request OTP Registrasi
- **URL**: `POST /api/v1/auth/register/request-otp`
- **Request Body**:
```json
{
  "email": "budi@example.com",
  "name": "Budi Santoso"
}
```
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "Kode OTP registrasi berhasil dikirim ke email Anda. Berlaku selama 5 menit.",
  "data": null
}
```

### 2. Verifikasi OTP Registrasi & Buat Akun
- **URL**: `POST /api/v1/auth/register/verify-otp`
- **Request Body**:
```json
{
  "email": "budi@example.com",
  "name": "Budi Santoso",
  "otpCode": "123456",
  "password": "Password123!"
}
```
- **Response Success (201 Created)**:
```json
{
  "success": true,
  "message": "Registrasi akun berhasil!",
  "data": {
    "token": "eyJhbGciOiJIUzI1Ni...",
    "user": {
      "id": "usr-uuid-1234",
      "email": "budi@example.com",
      "name": "Budi Santoso",
      "role": "USER",
      "isVerified": true,
      "createdAt": "2026-08-04T07:10:00Z",
      "updatedAt": "2026-08-04T07:10:00Z"
    }
  }
}
```

### 3. Login Menggunakan Password
- **URL**: `POST /api/v1/auth/login/password`
- **Request Body**:
```json
{
  "email": "budi@example.com",
  "password": "Password123!"
}
```
- **Response Success (200 OK)**: Mengembalikan `token` JWT dan objek `user`.

### 4. Login Menggunakan OTP (Tanpa Password)
1. **Request OTP Login**: `POST /api/v1/auth/login/request-otp`
   ```json
   { "email": "budi@example.com" }
   ```
2. **Verifikasi OTP Login**: `POST /api/v1/auth/login/verify-otp`
   ```json
   { "email": "budi@example.com", "otpCode": "654321" }
   ```
   *Response Success (200 OK)*: Mengembalikan `token` JWT dan objek `user`.

### 5. Forgot Password (Reset Kata Sandi)
1. **Request OTP Reset**: `POST /api/v1/auth/forgot-password/request-otp`
   ```json
   { "email": "budi@example.com" }
   ```
2. **Verifikasi OTP Reset**: `POST /api/v1/auth/forgot-password/verify-otp`
   ```json
   { "email": "budi@example.com", "otpCode": "112233" }
   ```
   *Response Success (200 OK)*: Mengembalikan `resetToken`.
3. **Eksekusi Reset Kata Sandi Baru**: `POST /api/v1/auth/forgot-password/reset`
   ```json
   {
     "resetToken": "eyJhbGciOiJIUzI1...",
     "newPassword": "PasswordBaruAman123!"
   }
   ```

### 6. Ambil Profil Saya
- **URL**: `GET /api/v1/auth/me`
- **Auth Required**: Ya (`Authorization: Bearer <token>`)
- **Response Success (200 OK)**: Mengembalikan objek `UserRecord`.

---

## 👤 Modul 3: Manajemen Akun & Profil (`/api/v1/user`)

Semua endpoint pada modul ini membutuhkan **Header**: `Authorization: Bearer <token>`

### 1. Update Profil (Nama & Nomor HP)
- **URL**: `PUT /api/v1/user/profile`
- **Request Body**:
```json
{
  "name": "Budi Santoso Terbaru",
  "phone": "081234567890"
}
```

### 2. Ubah Kata Sandi Dari Akun
- **URL**: `POST /api/v1/user/change-password`
- **Request Body**:
```json
{
  "oldPassword": "Password123!",
  "newPassword": "PasswordBaruUltraAman123!"
}
```

### 3. Ringkasan Dashboard User
- **URL**: `GET /api/v1/user/dashboard-summary`
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "Berhasil mengambil ringkasan statistik akun.",
  "data": {
    "totalTicketsBought": 4,
    "upcomingEventsCount": 2,
    "pastEventsCount": 2,
    "activeRefundsCount": 0
  }
}
```

### 4. Riwayat Pesanan Tiket Saya
- **URL**: `GET /api/v1/user/orders`
- **Query Parameter Opsional**: `?status=ISSUED` atau `?status=CHECKED_IN` atau `?status=REFUNDED`
- **Response Success (200 OK)**: Mengembalikan array berisi seluruh daftar pesanan milik pengguna terautentikasi.

### 5. Detail 1 Pesanan Saya
- **URL**: `GET /api/v1/user/orders/:orderCode` (Contoh: `/api/v1/user/orders/SYM-987654`)
- **Response Success (200 OK)**: Mengembalikan detail `OrderRecord` lengkap dengan data venue & QR Code.

### 6. Monitoring Refund Saya
- **URL**: `GET /api/v1/user/refunds`
- **Response Success (200 OK)**: Mengembalikan daftar seluruh permohonan refund milik pengguna beserta status (`PENDING`, `APPROVED`, `REJECTED`), catatan admin, dan nominal pengembalian.

---

## 🎟️ Modul 4: Pemesanan Tiket & Lookup (`/api/v1/orders`, `/api/v1/tickets`)

### 1. Checkout Tiket Konser (`POST /api/v1/orders`)
Endpoint ini menangani pemesanan tiket dengan fitur **Atomic Quota Deduction** dan **Row-Locking DB (`FOR UPDATE`)** untuk mencegah overbooking saat *ticket war*.

- **Fitur Otomatis Logged-In User**:
  - Apabila menyertakan Header `Authorization: Bearer <token>`, FE **TIDAK PERLU** mengirimkan `userName` dan `userEmail`. Sistem otomatis mengambil data akun user dan mengaitkan `userId` pada pesanan.
  - Untuk **Guest Checkout (Tanpa Token)**: Sertakan `userName` & `userEmail` secara manual.

- **Request Body (Logged-In User)**:
```json
{
  "eventId": "evt-1",
  "ticketCategoryId": "c1-vip",
  "quantity": 2
}
```

- **Request Body (Guest Checkout)**:
```json
{
  "eventId": "evt-1",
  "ticketCategoryId": "c1-vip",
  "quantity": 2,
  "userName": "Agus Wijaya",
  "userEmail": "agus@example.com"
}
```

- **Aturan Transaksi**:
  - `quantity`: Minimal 1 dan maksimal 4 tiket per transaksi.
  - Sisa kuota tiket (`remaining_quota`) akan berkurang secara otomatis.
  - Status pesanan diterbitkan sebagai `ISSUED` (terverifikasi), dan E-Ticket dikirimkan secara otomatis ke inbox Mailpit.

- **Response Success (201 Created)**:
```json
{
  "success": true,
  "message": "Simulasi Pembayaran Sandbox Berhasil & E-Ticket Terbit!",
  "data": {
    "id": "ord-uuid-5678",
    "orderCode": "SYM-849201",
    "userId": "usr-uuid-1234",
    "eventId": "evt-1",
    "eventTitle": "Symphony No. 5 in C minor",
    "artist": "Royal Philharmonic Orchestra & Jakarta Choral Society",
    "venue": "Aula Simfonia Jakarta",
    "date": "Sabtu, 18 April 2026 @ 19:30 WIB",
    "categoryName": "VIP Orchestral Pit",
    "quantity": 2,
    "totalPrice": 1500000,
    "userName": "Budi Santoso",
    "userEmail": "budi@example.com",
    "qrCode": "QR-SYM-849201",
    "status": "VERIFIED",
    "paymentMethod": "SANDBOX_PAYMENT",
    "createdAt": "2026-08-04T07:15:00Z"
  }
}
```

---

### 2. Public Ticket Lookup (Verifikasi Tiket Publik)
- **URL**: `GET /api/v1/tickets/lookup?code=SYM-849201`
- **Auth Required**: Tidak (Dapat diakses oleh siapa saja untuk verifikasi tiket cepat)
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "Tiket ditemukan",
  "data": {
    "id": "ord-uuid-5678",
    "orderCode": "SYM-849201",
    "eventTitle": "Symphony No. 5 in C minor",
    "artist": "Royal Philharmonic Orchestra & Jakarta Choral Society",
    "venue": "Aula Simfonia Jakarta",
    "date": "Sabtu, 18 April 2026 @ 19:30 WIB",
    "categoryName": "VIP Orchestral Pit",
    "quantity": 2,
    "totalPrice": 1500000,
    "userName": "Budi Santoso",
    "userEmail": "budi@example.com",
    "qrCode": "QR-SYM-849201",
    "status": "ISSUED",
    "paymentMethod": "SANDBOX_PAYMENT",
    "createdAt": "2026-08-04T07:15:00Z"
  }
}
```

---

## 🔄 Modul 5: Pengajuan Refund Publik (`/api/v1/refunds`)

Modul ini memungkinkan pembeli (User maupun Guest) untuk mengajukan pengembalian dana (refund) melalui verifikasi OTP Email.

### 1. Request OTP Verifikasi Refund
- **URL**: `POST /api/v1/refunds/request-otp`
- **Request Body**:
```json
{
  "orderCode": "SYM-849201",
  "userEmail": "budi@example.com"
}
```
- **Validasi Keamanan**:
  - Email harus cocok dengan pemegang tiket.
  - Tiket bertipe `CHECKED_IN` (sudah di-scan di venue), `REFUNDED`, `REFUND_REQUESTED`, atau `CANCELLED` tidak dapat di-refund.
- **Response Success (200 OK)**: Kode OTP 6-digit dikirim ke Mailpit (berlaku 10 menit).

### 2. Submit Formulir Pengajuan Refund
- **URL**: `POST /api/v1/refunds/submit`
- **Request Body**:
```json
{
  "orderCode": "SYM-849201",
  "userEmail": "budi@example.com",
  "otpCode": "847291",
  "bankName": "Bank Central Asia (BCA)",
  "accountNumber": "1234567890",
  "accountHolder": "Budi Santoso",
  "reason": "Halangan mendadak tugas luar kota"
}
```
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "Pengajuan refund berhasil dikirim. Tim Finance akan meninjau dan memproses pengembalian dana Anda.",
  "data": {
    "orderCode": "SYM-849201",
    "status": "PENDING",
    "bankName": "Bank Central Asia (BCA)",
    "accountHolder": "Budi Santoso"
  }
}
```

### 3. Cek Status Refund Publik
- **URL**: `POST /api/v1/refunds/status`
- **Request Body**:
```json
{
  "orderCode": "SYM-849201",
  "userEmail": "budi@example.com"
}
```
- **Response Success (200 OK)**: Menampilkan status pesanan beserta detail permohonan refund dan `adminNote`.

---

## 🛠️ Modul 6: Panel Manajemen Admin (`/api/v1/admin`)

### 1. Admin Login
- **URL**: `POST /api/v1/admin/login`
- **Request Body**:
```json
{
  "username": "admin",
  "password": "123"
}
```
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "Login Admin berhasil",
  "data": {
    "username": "admin",
    "token": "admin-session-token-symphoniatic-2026"
  }
}
```

---

### 2. Admin Dashboard & Metrics Analytics
- **URL**: `GET /api/v1/admin/dashboard`
- **Description**: Menyediakan analitik bisnis lengkap untuk dashboard admin secara real-time.
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "Metrik admin berhasil diambil",
  "data": {
    "totalRevenue": 15500000.00,
    "ticketsSold": 32,
    "remainingQuota": 245,
    "totalEvents": 4,
    "totalOrders": 18,
    "eventStats": [
      {
        "eventId": "evt-1",
        "title": "Symphony No. 5 in C minor",
        "revenue": 9000000.00,
        "ticketsSold": 15
      }
    ],
    "revenueTimeline": [
      { "month": "Mar", "revenue": 4500000.00, "tickets": 10 },
      { "month": "Apr", "revenue": 11000000.00, "tickets": 22 }
    ],
    "categoryDistribution": [
      { "name": "VIP Orchestral Pit", "value": 10500000.00 },
      { "name": "CAT 1 Grand Tier", "value": 5000000.00 }
    ],
    "recentOrders": [
      {
        "id": "ord-uuid-5678",
        "orderCode": "SYM-849201",
        "eventTitle": "Symphony No. 5 in C minor",
        "quantity": 2,
        "totalPrice": 1500000.00,
        "userName": "Budi Santoso",
        "status": "ISSUED",
        "createdAt": "2026-08-04T07:15:00Z"
      }
    ]
  }
}
```

---

### 3. Upload Gambar Event (Poster Upload)
- **URL**: `POST /api/v1/admin/upload`
- **Content-Type**: `multipart/form-data`
- **Form Data Field**: `image` (file binary gambar `.jpg`, `.png`, `.webp`)
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "File gambar berhasil diunggah",
  "data": {
    "url": "http://localhost:8082/uploads/1722000000000.jpg"
  }
}
```

---

### 4. Manajemen Event Konser (Admin Event CRUD)

#### A. Tambah Event Konser Baru (Beserta Kategori Tiket)
- **URL**: `POST /api/v1/admin/events`
- **Request Body**:
```json
{
  "title": "Simfoni Mahakarya Nusantara 2026",
  "subtitle": "Konser Megah Lintas Generasi",
  "artist": "Orkestra Youth Indonesia",
  "conductor": "Maestro Addie MS",
  "venue": "Aula Simfonia Jakarta",
  "address": "Jl. Industri Blok B14 No.1, Kemayoran, Jakarta Pusat",
  "date": "Sabtu, 15 November 2026",
  "time": "19:30 WIB",
  "openGate": "18:00 WIB",
  "category": "NUSANTARA SYMPHONY",
  "categoryBadgeColor": "bg-blue-900/80 text-blue-200 border-blue-500/40",
  "image": "http://localhost:8082/uploads/1722000000000.jpg",
  "audioUrl": "",
  "organizer": "SymphoniaTic Management",
  "description": "Pagelaran musik simfoni akbar memperingati hari musik...",
  "rundown": [
    { "time": "18:00 WIB", "activity": "Open Gate & Check In" },
    { "time": "19:30 WIB", "activity": "Pertunjukan Utama Pembuka" }
  ],
  "categories": [
    { "name": "VVIP Royal", "price": 1500000, "quota": 50 },
    { "name": "VIP Platinum", "price": 1000000, "quota": 100 },
    { "name": "Reguler Stalls", "price": 500000, "quota": 200 }
  ]
}
```
- **Response Success (201 Created)**: Mengembalikan data event dan daftar kategori tiket yang baru dibuat.

#### B. Update Detail Event Konser
- **URL**: `PUT /api/v1/admin/events/:id` (Contoh: `/api/v1/admin/events/evt-1`)
- **Request Body**: Sama seperti `POST /api/v1/admin/events` (tanpa array `categories`).

#### C. Tutup / Buka Penjualan Tiket Konser
- **URL**: `PATCH /api/v1/admin/events/:id/toggle-close`
- **Response Success (200 OK)**:
```json
{
  "success": true,
  "message": "Penjualan tiket konser berhasil ditutup",
  "data": {
    "eventId": "evt-1",
    "isClosed": true
  }
}
```

#### D. Hapus Event Konser
- **URL**: `DELETE /api/v1/admin/events/:id`
- **Response Success (200 OK)**: Menghapus event beserta seluruh kategori tiket terkait secara *cascade*.

---

### 5. Manajemen Kategori Tiket (Admin Category CRUD)

#### A. Tambah Kategori Tiket ke Event
- **URL**: `POST /api/v1/admin/events/:id/categories` (Contoh: `/api/v1/admin/events/evt-1/categories`)
- **Request Body**:
```json
{
  "name": "Super VVIP Box",
  "price": 2000000,
  "quota": 20
}
```

#### B. Update Kategori Tiket
- **URL**: `PUT /api/v1/admin/categories/:id` (Contoh: `/api/v1/admin/categories/c1-vip`)
- **Request Body**:
```json
{
  "name": "VIP Orchestral Pit Revised",
  "price": 800000,
  "quota": 20
}
```
*Sistem otomatis menyesuaikan `remaining_quota` secara proporsional sesuai selisih quota baru.*

#### C. Hapus Kategori Tiket
- **URL**: `DELETE /api/v1/admin/categories/:id`

---

### 6. Kelola Transaksi Pesanan (Admin Orders Management)

#### A. Ambil Semua Transaksi Pesanan Tiket
- **URL**: `GET /api/v1/admin/orders`
- **Query Params Opsional**:
  - `?search=budi`: Cari berdasarkan kode order, nama pembeli, email, atau judul event.
  - `?status=ISSUED`: Filter status (`ISSUED`, `VERIFIED`, `CHECKED_IN`, `REFUNDED`, `CANCELLED`).
- **Response Success (200 OK)**: Mengembalikan array `OrderRecord[]`.

#### B. Update Status Transaksi Pesanan Tiket
- **URL**: `PATCH /api/v1/admin/orders/:id/status`
- **Request Body**:
```json
{
  "status": "CHECKED_IN"
}
```
*Catatan Penting*: Jika `status` diubah menjadi `"REMINDED"`, sistem otomatis mengirimkan email pengingat pertunjukan konser ke email pembeli via Mailpit.

---

### 7. Kelola Permohonan Refund (Admin Refund Management)

#### A. Ambil Semua Permohonan Refund
- **URL**: `GET /api/v1/admin/refunds`
- **Response Success (200 OK)**: Mengembalikan daftar seluruh permohonan refund pembeli lengkap dengan data rekening bank dan detail tiket.

#### B. Setujui atau Tolak Permohonan Refund
- **URL**: `PATCH /api/v1/admin/refunds/:id/status`
- **Request Body (Menyetujui Refund)**:
```json
{
  "status": "APPROVED",
  "adminNote": "Dana sebesar Rp 1.500.000 telah ditransfer ke rekening BCA Anda."
}
```
- **Request Body (Menolak Refund)**:
```json
{
  "status": "REJECTED",
  "adminNote": "Pengajuan ditolak karena melewati batas waktu H-3 konser."
}
```
- **Logika Bisnis Otomatis di Backend**:
  - **Jika `APPROVED` / `COMPLETED`**:
    1. Status pesanan di database otomatis berubah menjadi `REFUNDED`.
    2. Kuota tiket (`remaining_quota`) pada kategori terkait **otomatis direstock (dikembalikan)**.
    3. Email konfirmasi persetujuan refund dikirimkan ke pembeli.
  - **Jika `REJECTED`**:
    1. Status pesanan dikembalikan menjadi `VERIFIED`.
    2. Email notifikasi penolakan beserta alasan admin dikirimkan ke pembeli.

---

## 🛡️ Aturan Keamanan OTP & Rate Limiting

Sistem Autentikasi dan Refund menggunakan mekanisme OTP teramankan dengan aturan berikut:

- ⏱️ **Batas Waktu Expiry**: Kode OTP berlaku selama **5 menit** (Auth) / **10 menit** (Refund).
- ⏳ **Resend Cooldown**: Pengiriman OTP ulang ditahan **60 detik**. Request sebelum 60 detik mengembalikan HTTP Status `429 Too Many Requests`.
- ❌ **Max Attempt Limit**: Maksimal **5 kali salah** memasukkan OTP sebelum kode OTP secara otomatis dihanguskan (invalidated).

---

## 🗄️ Skema Database PostgreSQL (Entity Relationship)

Sistem menggunakan skema relational pada PostgreSQL dengan tabel utama:

1. **`events`**: Menyimpan master data pertunjukan konser simfoni, venue, tanggal, konduktor, audio preview, & rundown (`JSONB`).
2. **`ticket_categories`**: Menyimpan kategori tiket (VIP, CAT 1, dll), harga, total kuota, & `remaining_quota` per event.
3. **`orders`**: Menyimpan data transaksi pesanan tiket, status (`ISSUED`, `CHECKED_IN`, `REFUNDED`), total harga, & QR Code.
4. **`refund_requests`**: Menyimpan permohonan pengembalian dana, data rekening penerima, alasan, status refund, & OTP code.
5. **`users`**: Menyimpan akun pengguna terdaftar, password hash (Bcrypt), nama, nomor hp, & role (`USER`/`ADMIN`).
6. **`auth_otps`**: Menyimpan log OTP transaksi autentikasi (purpose: `REGISTER`, `LOGIN`, `FORGOT_PASSWORD`).

---

## ✉️ Pengujian Email & Mailpit

Seluruh pengiriman email pada environment lokal diarahkan ke server **Mailpit**:
- Buka browser di **`http://localhost:8025`**.
- Seluruh pesan email berikut dapat ditinjau secara visual:
  - 📩 **Email Kode OTP Registrasi / Login / Reset Password**
  - 🎟️ **Email E-Ticket dengan attachment QR Code & detail kursi**
  - ⏰ **Email Pengingat Event Konser (Event Reminder)**
  - 💰 **Email Notifikasi Status Refund (Approved / Rejected)**
