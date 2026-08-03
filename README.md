# 🎼 SymphoniaTic Backend REST API (Dokumentasi Front-End Integration)

Dokumentasi resmi **SymphoniaTic Backend REST API** yang dirancang khusus untuk tim **Front-End (FE)**. Dokumentasi ini memuat seluruh daftar endpoint, format request/response JSON, alur Autentikasi (OTP & Password), Autofill Checkout, Riwayat Pesanan User, Manajemen Profil, dan Pengujian Mailpit.

---

## ⚡ Ringkasan Server & Setup Lokal

| Environment | URL / Alamat | Keterangan |
| :--- | :--- | :--- |
| **API Base URL** | `http://localhost:8082/api/v1` | Prefix utama seluruh endpoint API |
| **Mailpit Dashboard** | `http://localhost:8025` | Interface browser untuk melihat email OTP & E-Ticket |
| **Mailpit SMTP** | `localhost:1025` | Port SMTP lokal |
| **Auth Header** | `Authorization: Bearer <token>` | Header wajib untuk endpoint akun user terautentikasi |

---

## 🚀 Panduan Memulai Backend & Mailpit

### 1. Jalankan PostgreSQL & Mailpit via Docker
```bash
docker run -d --name symphoniatic-db -p 5432:5432 postgres:alpine
docker run -d --name mailpit -p 8025:8025 -p 1025:1025 axllent/mailpit
```

### 2. Jalankan Server Go Backend
```bash
go run ./cmd/server
```
*Server API aktif di `http://localhost:8082`.*

---

## 🔑 1. Autentikasi User (Register, Login, Forgot Password)

### A. Alur Registrasi Akun Baru (OTP Email + Password)

1. **Request OTP Registrasi**: `POST /api/v1/auth/register/request-otp`
   ```json
   {
     "email": "user@example.com",
     "name": "Budi Santoso"
   }
   ```
2. **Verifikasi OTP Registrasi & Password**: `POST /api/v1/auth/register/verify-otp`
   ```json
   {
     "email": "user@example.com",
     "name": "Budi Santoso",
     "otpCode": "123456",
     "password": "Password123!"
   }
   ```
   *Response (201 Created)*: Mengembalikan `token` JWT dan objek `user`.

---

### B. Alur Login

- **Login Password**: `POST /api/v1/auth/login/password`
  ```json
  { "email": "user@example.com", "password": "Password123!" }
  ```
- **Login OTP (Tanpa Password)**:
  1. `POST /api/v1/auth/login/request-otp` -> `{ "email": "user@example.com" }`
  2. `POST /api/v1/auth/login/verify-otp` -> `{ "email": "user@example.com", "otpCode": "654321" }`

---

### C. Alur Forgot Password (Reset Kata Sandi)

1. `POST /api/v1/auth/forgot-password/request-otp` -> `{ "email": "user@example.com" }`
2. `POST /api/v1/auth/forgot-password/verify-otp` -> `{ "email": "user@example.com", "otpCode": "112233" }` *(Mendapatkan `resetToken`)*
3. `POST /api/v1/auth/forgot-password/reset`
   ```json
   {
     "resetToken": "eyJhbGciOiJIUzI1...",
     "newPassword": "PasswordBaru123!"
   }
   ```

---

## 👤 2. Manajemen Akun & Profil User (`/api/v1/user`)

Semua endpoint di bawah ini membutuhkan **Header**: `Authorization: Bearer <token>`

### A. Get Profil Pengguna
- **`GET /api/v1/auth/me`**

### B. Update Profil (Nama & Nomor HP)
- **`PUT /api/v1/user/profile`**
- **Request Body**:
  ```json
  {
    "name": "Budi Santoso Terbaru",
    "phone": "081234567890"
  }
  ```

### C. Ubah Kata Sandi (Dari Dalam Akun)
- **`POST /api/v1/user/change-password`**
- **Request Body**:
  ```json
  {
    "oldPassword": "Password123!",
    "newPassword": "PasswordBaruUltraAman123!"
  }
  ```

### D. Ringkasan Dashboard User (Statistik Ringkas)
- **`GET /api/v1/user/dashboard-summary`**
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

---

## 🎟️ 3. Pemesanan Tiket & Riwayat Pesanan User

### A. Autofill Checkout Tiket (`POST /api/v1/orders`)
- **Fitur Otomatis untuk User Logged-In**:
  - Apabila menyertakan Header `Authorization: Bearer <token>`, FE **TIDAK PERLU** mengirimkan `userName` dan `userEmail`. Sistem akan mengisinya secara otomatis dari data akun user dan mencatat `userId` pada pesanan.
  - Untuk Guest (Tanpa Token): Tetap kirimkan `userName` & `userEmail` manual.
- **Request Body (Logged-In User)**:
  ```json
  {
    "eventId": "evt-1",
    "ticketCategoryId": "cat-1-1",
    "quantity": 2
  }
  ```

### B. Riwayat Pesanan Saya
- **`GET /api/v1/user/orders`**
- **Header**: `Authorization: Bearer <token>`
- **Query Params Opsional**: `?status=ISSUED` atau `?status=REFUNDED`
- **Response**: Mengembalikan daftar seluruh tiket konser milik pengguna terautentikasi (dilengkapi QR Code & status transaksi).

### C. Detail 1 Pesanan Saya
- **`GET /api/v1/user/orders/:orderCode`** (Contoh: `/api/v1/user/orders/SYM-987654`)

---

## 🔄 4. Monitoring Pengajuan Refund User

- **`GET /api/v1/user/refunds`**
- **Header**: `Authorization: Bearer <token>`
- **Response**: Menampilkan riwayat status pengajuan refund pengguna (`PENDING`, `APPROVED`, `REJECTED`) beserta catatan admin dan jumlah nominal refund.

---

## 🛡️ Catatan Aturan Keamanan OTP untuk FE

- ⏱️ **Batas Waktu Expiry**: Kode OTP berlaku **5 menit**.
- ⏳ **Resend Cooldown**: Pengiriman OTP ulang ditahan **60 detik** (Status HTTP `429 Too Many Requests`).
- ❌ **Max Attempt Limit**: Maksimal **5 kali salah** memasukkan OTP sebelum kode hangus.
