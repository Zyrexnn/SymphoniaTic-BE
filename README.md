# 🎼 SymphoniaTic Backend REST API (Dokumentasi Front-End Integration)

Dokumentasi resmi **SymphoniaTic Backend REST API** yang dirancang khusus untuk tim **Front-End (FE)**. Dokumentasi ini memuat seluruh daftar endpoint, format request/response JSON, alur Autentikasi (OTP & Password), dan panduan Mailpit SMTP.

---

## ⚡ Ringkasan Server & Setup Lokal

| Environment | URL / Alamat | Keterangan |
| :--- | :--- | :--- |
| **API Base URL** | `http://localhost:8082/api/v1` | Prefix utama seluruh endpoint API |
| **Mailpit Dashboard** | `http://localhost:8025` | Interface browser untuk melihat email OTP & E-Ticket |
| **Mailpit SMTP** | `localhost:1025` | Port SMTP lokal |
| **Auth Header** | `Authorization: Bearer <token>` | Header wajib untuk endpoint yang membutuhkan login |

---

## 🚀 Panduan Memulai Backend & Mailpit

### 1. Jalankan PostgreSQL & Mailpit via Docker
```bash
docker run -d --name symphoniatic-postgres -e POSTGRES_USER=symphoniatic -e POSTGRES_PASSWORD=symphoniatic_secret -e POSTGRES_DB=symphoniatic_db -p 5432:5432 postgres:16-alpine
docker run -d --name symphoniatic-mailpit -p 8025:8025 -p 1025:1025 axllent/mailpit
```

### 2. Jalankan Server Go Backend
```bash
go run ./cmd/server
```
*Server API aktif di `http://localhost:8082`.*

---

## 🔑 1. Autentikasi User (Register, Login, Forgot Password)

### A. Alur Registrasi Akun Baru (OTP Email + Password)

1. **Request OTP Registrasi**
   - **`POST /api/v1/auth/register/request-otp`**
   - **Request Body**:
     ```json
     {
       "email": "user@example.com",
       "name": "Budi Santoso"
     }
     ```
   - **Response Success (200 OK)**:
     ```json
     {
       "success": true,
       "message": "Kode OTP registrasi berhasil dikirim ke email Anda. Berlaku selama 5 menit."
     }
     ```

2. **Verifikasi OTP Registrasi & Buat Password**
   - **`POST /api/v1/auth/register/verify-otp`**
   - **Request Body**:
     ```json
     {
       "email": "user@example.com",
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
         "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6...",
         "user": {
           "id": "usr-uuid-1234",
           "email": "user@example.com",
           "name": "Budi Santoso",
           "role": "USER",
           "isVerified": true
         }
       }
     }
     ```

---

### B. Alur Login (Password & OTP)

#### Method 1: Login Standar (Password)
- **`POST /api/v1/auth/login/password`**
- **Request Body**:
  ```json
  {
    "email": "user@example.com",
    "password": "Password123!"
  }
  ```
- **Response Success (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Login berhasil!",
    "data": {
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6...",
      "user": {
        "id": "usr-uuid-1234",
        "email": "user@example.com",
        "name": "Budi Santoso",
        "role": "USER"
      }
    }
  }
  ```

#### Method 2: Login via OTP Email (Tanpa Password)
1. **Request OTP Login**: `POST /api/v1/auth/login/request-otp`
   ```json
   { "email": "user@example.com" }
   ```
2. **Verify OTP Login**: `POST /api/v1/auth/login/verify-otp`
   ```json
   { "email": "user@example.com", "otpCode": "654321" }
   ```

---

### C. Alur Forgot Password (Lupa Kata Sandi)

1. **Request OTP Reset Password**: `POST /api/v1/auth/forgot-password/request-otp`
   ```json
   { "email": "user@example.com" }
   ```

2. **Verifikasi OTP Reset Password**: `POST /api/v1/auth/forgot-password/verify-otp`
   ```json
   { "email": "user@example.com", "otpCode": "112233" }
   ```
   *Response mendapatkan `resetToken` (berlaku 15 menit)*:
   ```json
   {
     "success": true,
     "message": "Verifikasi OTP berhasil!",
     "data": {
       "resetToken": "eyJhbGciOiJIUzI1..."
     }
   }
   ```

3. **Setel Kata Sandi Baru**: `POST /api/v1/auth/forgot-password/reset`
   ```json
   {
     "resetToken": "eyJhbGciOiJIUzI1...",
     "newPassword": "PasswordBaru123!"
   }
   ```

---

### D. Get Profile User Terautentikasi
- **`GET /api/v1/auth/me`**
- **Header**: `Authorization: Bearer <token>`
- **Response Success (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Berhasil mengambil profil pengguna.",
    "data": {
      "id": "usr-uuid-1234",
      "email": "user@example.com",
      "name": "Budi Santoso",
      "phone": "",
      "role": "USER",
      "isVerified": true
    }
  }
  ```

---

## 🎟️ 2. Konser & Pembelian Tiket (Events & Orders)

### A. Ambil Semua Daftar Konser
- **`GET /api/v1/events`**
- **Response**: Menampilkan array daftar konser lengkap dengan kategori tiket & sisa kuota.

### B. Ambil Detail Konser
- **`GET /api/v1/events/:id`** (Contoh: `/api/v1/events/evt-1`)

### C. Pemesanan Tiket (Guest Checkout)
- **`POST /api/v1/orders`**
- **Request Body**:
  ```json
  {
    "eventId": "evt-1",
    "ticketCategoryId": "cat-1-1",
    "quantity": 2,
    "userName": "Budi Santoso",
    "userEmail": "user@example.com"
  }
  ```
- **Response (201 Created)**: Menerbitkan Kode Pesanan (`orderCode`: `SYM-XXXXXX`) dan QR Code, serta mengirimkan E-Ticket HTML ke Mailpit.

### D. Lookup Tiket Publik
- **`GET /api/v1/tickets/lookup?code=SYM-123456`**

---

## 🔄 3. Pengajuan Refund Tiket (2-Step OTP)

1. **Request Refund OTP**: `POST /api/v1/refunds/request-otp`
   ```json
   { "orderCode": "SYM-123456", "userEmail": "user@example.com" }
   ```
2. **Submit Pengajuan Refund**: `POST /api/v1/refunds/submit`
   ```json
   {
     "orderCode": "SYM-123456",
     "userEmail": "user@example.com",
     "otpCode": "998877",
     "bankName": "BCA",
     "accountNumber": "1234567890",
     "accountHolder": "Budi Santoso",
     "reason": "Halangan mendadak"
   }
   ```
3. **Cek Status Refund**: `POST /api/v1/refunds/status`

---

## 🛡️ Catatan Aturan Keamanan OTP untuk FE

- ⏱️ **Batas Waktu Expiry**: Kode OTP berlaku **5 menit**.
- ⏳ **Resend Cooldown**: Pengiriman OTP ulang ditahan **60 detik**. Jika kurang dari 60 detik, server mengembalikan status HTTP `429 Too Many Requests`.
- ❌ **Max Attempt Limit**: Maksimal **5 kali salah** memasukkan OTP. Pada percobaan ke-5, OTP otomatis hangus dan pengguna harus meminta kode baru.
