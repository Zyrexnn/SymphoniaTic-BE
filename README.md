# SymphoniaTic Backend (Go + PostgreSQL + Fiber)

REST API backend performa tinggi untuk platform pemesanan tiket konser SymphoniaTic.

## 🚀 Fitur Utama
- **Guest Checkout (Tanpa Login)**: Pembelian tiket instan dengan alokasi kuota aman.
- **Row Locking (`FOR UPDATE`)**: Transaksi atomic untuk mencegah overbooking saat *ticket war*.
- **PostgreSQL Native (Docker Container)**: Penyimpanan relational data terstruktur.
- **Lookup Tiket Publik**: Pencarian tiket & status hanya dengan Kode Pemesanan/Invoice (`SYM-XXXXXX`).
- **Simulasi Sandbox Payment**: Transaksi auto-verified untuk demo/testing.

## 🛠️ Cara Menjalankan

1. **Jalankan PostgreSQL via Docker**:
   ```bash
   docker run -d --name symphoniatic-postgres -e POSTGRES_USER=symphoniatic -e POSTGRES_PASSWORD=symphoniatic_secret -e POSTGRES_DB=symphoniatic_db -p 5432:5432 postgres:16-alpine
   ```

2. **Eksekusi File Query SQL (`setup.sql`)**:
   - Jika ingin mengimpor tabel & data awal secara manual untuk tim kerja:
   ```bash
   docker exec -i symphoniatic-postgres psql -U symphoniatic -d symphoniatic_db < setup.sql
   ```

3. **Jalankan Backend Go**:
   ```bash
   go run main.go
   ```
   API akan berjalan di `http://localhost:8082`.

## 🌐 Endpoint API (`/api/v1`)
- `GET /api/v1/events` - Daftar konser & kategori tiket + kuota
- `GET /api/v1/events/:id` - Detail konser
- `POST /api/v1/orders` - Checkout pesanan tiket (Guest)
- `GET /api/v1/tickets/lookup?code=SYM-123456` - Cek tiket via kode pesanan
- `GET /api/v1/admin/dashboard` - Metrik admin
