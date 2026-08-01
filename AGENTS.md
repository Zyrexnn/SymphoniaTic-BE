# Backend — SymphoniaTic (Go + Fiber + PostgreSQL)

REST API untuk platform pemesanan tiket konser. Bahasa utama: **Bahasa Indonesia** (log, komentar, respons error).

## Stack
- Go 1.25 + Fiber v2
- PostgreSQL (lib/pq), koneksi native `database/sql`
- Docker Postgres 16 + `setup.sql`
- Env config via `github.com/joho/godotenv` (`.env`)

## Menjalankan
```bash
docker run -d --name symphoniatic-postgres \
  -e POSTGRES_USER=symphoniatic -e POSTGRES_PASSWORD=symphoniatic_secret \
  -e POSTGRES_DB=symphoniatic_db -p 5432:5432 postgres:16-alpine
go run ./cmd/server   # server di http://localhost:8082 (default PORT=8082)
```
Server default port: `8082` (lihat `cmd/server/main.go`). Prefix semua route: `/api/v1`.

## Struktur
- `cmd/server/main.go` — entrypoint, wiring route & middleware (CORS, logger)
- `controllers/controllers.go` — semua handler HTTP
- `database/database.go` — koneksi, `initSchema` (DDL), `seedInitialData`
- `models/models.go` — struct data/request/response
- `services/mailer.go` — kirim email (mis. OTP refund)

## Konvensi
- Bahasa Indonesia untuk string log, error message, dan nama endpoint yang ramah pengguna.
- ID record: string UUID custom (mis. `evt-1`, `cat-1-1`, order `SYM-XXXXXX`).
- Skema dibuat otomatis di `initSchema`; jangan edit `setup.sql` manual kecuali menambah tabel — pastikan DDL ada di `initSchema` juga.
- Transaksi tiket pakai row locking `FOR UPDATE` untuk cegah overbooking.
- Sandbox payment: status order auto `VERIFIED`, `payment_method=SANDBOX_PAYMENT`.
- CORS allowlist localhost dev (jangan buka semua origin).

## Endpoint utama (`/api/v1`)
- `GET /events`, `GET /events/:id`
- `POST /orders`
- `GET /tickets/lookup?code=SYM-123456`
- `POST /refunds/request-otp`, `/refunds/submit`, `/refunds/status`
- Admin (`/admin`): `POST /login`, `GET /dashboard`, CRUD events/categories/orders/refunds

## Perintah
```bash
go build ./...   # kompilasi
go run ./cmd/server
```
Tidak ada test suite; verifikasi via `go build` dan curl endpoint.
