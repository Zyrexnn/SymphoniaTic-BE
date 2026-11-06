# AGENTS.md: Backend (Go + Fiber + PostgreSQL)

> Operational instructions for AI coding agents working on the SymphoniaTic backend REST API.

---

## 1. Stack & Tooling

- **Language & Runtime**: Go 1.25+
- **HTTP Framework**: Fiber v2 (`github.com/gofiber/fiber/v2`)
- **Database Driver**: Native `database/sql` with PostgreSQL driver (`github.com/lib/pq`), without an ORM
- **Documentation**: Swagger / OpenAPI (`github.com/gofiber/swagger`, generated files in `docs/`)
- **Environment**: `github.com/joho/godotenv` (`.env`)
- **Mailing**: Native `net/smtp` connected to Mailpit (`localhost:1025`)

---

## 2. Setup & Development Commands

```bash
# Download and verify Go dependencies
go mod download
go mod tidy

# Run backend development server (default port 8082)
go run ./cmd/server

# Compile and check for type and syntax errors
go build ./...

# Regenerate Swagger documentation when route annotations change
swag init -g cmd/server/main.go -o docs
```

---

## 3. Directory Layout & Boundaries

```text
be/
├── cmd/server/main.go   # Entrypoint, CORS allowlist, middleware, routing table
├── controllers/         # HTTP handlers and request validation
├── database/            # Connection pooling, DDL initSchema(), seedInitialData()
├── models/              # Struct definitions for DB entities, requests, and APIResponse
├── services/            # External services (mailer.go connecting to Mailpit)
├── utils/               # JWT token generation/validation, password hashing
├── docs/                # Generated Swagger OpenAPI artifacts
├── setup.sql            # PostgreSQL DDL and initial seed data
└── uploads/             # Static uploaded files (served at /uploads)
```

---

## 4. Coding Standards & Mandatory Rules

### A. Language & Communication Idiom
- **Logs, comments, and client-facing error messages MUST be in Bahasa Indonesia**.
- Examples: `"Berhasil mengambil data konser"`, `"Gagal memproses pesanan tiket"`, `"Kategori tiket tidak ditemukan"`.

### B. Standard API Response Pattern
Always return responses wrapped in `models.APIResponse`:
```go
// Success response
return ctx.Status(fiber.StatusOK).JSON(models.APIResponse{
    Success: true,
    Message: "Data berhasil diperbarui",
    Data:    payload,
})

// Error response
return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
    Success: false,
    Message: "Format email tidak valid",
})
```

### C. Concurrency Safety & Ticket War Transactions
- During ticket checkout (`CreateOrder`), always wrap the operation inside a database transaction (`DB.Begin()`).
- Always lock the ticket quota row using **`SELECT ... FOR UPDATE`** before decrementing `remaining_quota`:
  ```sql
  SELECT remaining_quota FROM ticket_categories WHERE id = $1 FOR UPDATE;
  ```
- If remaining quota is insufficient, roll back immediately and return HTTP 400.

### D. Database Schema Management
- Tables are created automatically by `database.initSchema()` on server boot.
- When adding tables or columns, update both:
  1. `initSchema()` in `database/database.go`
  2. `setup.sql`

### E. CORS Configuration
- CORS is configured in `cmd/server/main.go` using the dynamic matcher `AllowOriginsFunc` which allows local origins (`http://localhost:*`, `http://127.0.0.1:*`).
- Never replace this with wildcard `AllowOrigins: "*"` because credentials mode requires explicit origins.

---

## 5. Anti-Slop Discipline for Backend (`antislop-code`)

Apply antislop discipline to backend code to prevent generic AI patterns:

1. **Code Comment Hygiene**:
   - **FORBIDDEN**: Decorative ASCII separators such as `// ========================` or `// ------- ROUTES -------`. Use normal vertical spacing between functions.
   - **FORBIDDEN**: Comments that simply repeat function signatures or code statements (for example, `// GetEvents retrieves events` above `func GetEvents`, or `// Loop order items` above `for range`).
   - **FORBIDDEN**: Decorative emojis in comments (`// 🚀 Fast query`, `// ✅ Validation success`).
   - **FORBIDDEN**: Numbered workflow narration (`// Step 1: Query database`, `// Step 2: Validate`). Let the code structure speak for itself.
   - **REQUIRED**: Write comments only to explain technical reasons (**WHY**), such as transaction isolation rationale, refund calculation formulas, or security sanitization constraints.

2. **Data & Endpoint Integrity**:
   - **FORBIDDEN**: Creating dummy mock endpoints that return success without real business logic (Craftsmanship C-2).
   - **FORBIDDEN**: Populating database seeds with nonsensical placeholder data (*"Concert test 123"*, 0 price). All orchestra concert data must realistically reflect classical performances.

---

## 6. Guardrails (Never-Do List)

- **NEVER** introduce an ORM (e.g. GORM, Ent, Bun). Maintain native `database/sql` queries.
- **NEVER** change the base API prefix from `/api/v1`.
- **NEVER** commit plain text passwords or secrets to Git. Read credentials through `os.Getenv()`.
- **NEVER** omit database transaction rollback on error paths (`defer tx.Rollback()`).
- **NEVER** write client error responses in English (the client contract expects Indonesian messages).

---

## 7. Verification & Definition of Done

Before declaring any backend task complete, execute:

1. **Compilation Check**:
   ```bash
   go build ./...
   ```
   *Must exit with code 0 and produce no build errors.*

2. **Endpoint Smoke Test**:
   ```bash
   # Test public events endpoint
   curl -s http://localhost:8082/api/v1/events | grep '"success":true'
   ```
