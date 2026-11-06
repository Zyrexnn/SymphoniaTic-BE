# SymphoniaTic Backend REST API

Official REST API documentation for the SymphoniaTic classical concert ticketing platform. Covers all public, user authentication, customer account, checkout, public refund, and admin management endpoints, including JSON schemas, OTP security rules, transactional row-locking mechanisms, and local Mailpit email inspection.

---

## Server Summary & Local Setup

| Environment | URL / Address | Notes |
| :--- | :--- | :--- |
| **API Base URL** | `http://localhost:8082/api/v1` | Primary prefix for all REST endpoints |
| **Server Engine** | Go + Fiber v2 | Native web framework |
| **Database** | PostgreSQL 16 | Default port: `5432` |
| **Mailpit Dashboard** | `http://localhost:8025` | Webmail UI to inspect OTP, e-tickets, and notifications |
| **Mailpit SMTP** | `localhost:1025` | Local SMTP port for automated transactional email delivery |
| **Auth Header (User)** | `Authorization: Bearer <user_jwt_token>` | Required header for customer account and profile routes |
| **Auth Header (Admin)** | `Authorization: Bearer <admin_token>` | Admin session token for management routes |

---

## Getting Started with Backend & Mailpit

### 1. Environment Variables Configuration (`.env`)
Create a `.env` file in the `be/` folder with the following configuration:
```env
PORT=8082
DB_HOST=localhost
DB_PORT=5432
DB_USER=symphoniatic
DB_PASSWORD=symphoniatic_secret
DB_NAME=symphoniatic_db
DB_SSLMODE=disable
CORS_ORIGIN=http://localhost:4321,http://localhost:3000
API_BASE_URL=http://localhost:8082
JWT_SECRET=symphoniatic-super-secret-jwt-key-2026

# Mailpit SMTP Configuration
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_SENDER_EMAIL=noreply@symphoniatic.com
SMTP_SENDER_NAME=SymphoniaTic E-Ticket System

# Admin and Gate Officer Credentials
ADMIN_USERNAME=admin
ADMIN_PASSWORD=123
PETUGAS_USERNAME=petugas
PETUGAS_PASSWORD=123
```

### 2. Run PostgreSQL & Mailpit via Docker
```bash
# Start background database and mailpit services from repo root
docker compose up -d db mailpit
```

### 3. Database Schema and Seed Data
The database schema and initial concert catalog are initialized automatically on boot by `database.initSchema()`. For manual execution or raw database inspection:
```bash
psql -h localhost -U symphoniatic -d symphoniatic_db -f setup.sql
```

### 4. Run Go Backend Server
```bash
cd be
go run ./cmd/server
```
API server is active at `http://localhost:8082`. Swagger docs are available at `http://localhost:8082/swagger/index.html`.

---

## Standard API Response Envelope

All endpoints in the SymphoniaTic REST API return consistent JSON structures using the `APIResponse` envelope:

### Successful Request (200 OK / 201 Created)
```json
{
  "success": true,
  "message": "User-friendly status message in Bahasa Indonesia",
  "data": { ... }
}
```

### Failed Request (400 / 401 / 403 / 404 / 429 / 500)
```json
{
  "success": false,
  "message": "User-friendly error message in Bahasa Indonesia",
  "error": "Internal technical detail (optional / debug only)"
}
```

---

## Endpoint Quick Reference Table

### 1. Events & Concerts (Public)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/events` | List all concert events with categories and rundowns | Public |
| `GET` | `/api/v1/events/:id` | Retrieve single concert event by ID | Public |

### 2. User Authentication (`/api/v1/auth`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/auth/register/request-otp` | Request OTP for new account registration | Public |
| `POST` | `/api/v1/auth/register/verify-otp` | Verify registration OTP and set account password | Public |
| `POST` | `/api/v1/auth/login/password` | Login using email and password | Public |
| `POST` | `/api/v1/auth/login/request-otp` | Request OTP for passwordless login | Public |
| `POST` | `/api/v1/auth/login/verify-otp` | Verify login OTP and receive JWT token | Public |
| `POST` | `/api/v1/auth/forgot-password/request-otp` | Request password reset OTP | Public |
| `POST` | `/api/v1/auth/forgot-password/verify-otp` | Verify reset OTP and receive reset token | Public |
| `POST` | `/api/v1/auth/forgot-password/reset` | Set new password using reset token | Public |
| `GET` | `/api/v1/auth/me` | Fetch authenticated user profile | Bearer Token |

### 3. User Account Management (`/api/v1/user`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `PUT` | `/api/v1/user/profile` | Update user name and phone number | Bearer Token |
| `POST` | `/api/v1/user/change-password` | Change password from inside active session | Bearer Token |
| `GET` | `/api/v1/user/dashboard-summary` | Summary metrics: tickets, events, refunds | Bearer Token |
| `GET` | `/api/v1/user/orders` | Order history for authenticated user (filter: `?status=`) | Bearer Token |
| `GET` | `/api/v1/user/orders/:orderCode` | Detail of a specific user order | Bearer Token |
| `GET` | `/api/v1/user/refunds` | Refund history and status tracking | Bearer Token |

### 4. Ticket Orders & Lookup (`/api/v1/orders`, `/api/v1/tickets`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/orders` | Ticket checkout with row locking (autofills if authenticated) | Public / User |
| `GET` | `/api/v1/tickets/lookup` | Public lookup by order code (`?code=SYM-xxx`) | Public |

### 5. Public Refunds (`/api/v1/refunds`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/refunds/request-otp` | Request OTP to verify refund application | Public |
| `POST` | `/api/v1/refunds/submit` | Submit refund form (OTP + bank account) | Public |
| `POST` | `/api/v1/refunds/status` | Check refund status by order code and email | Public |

### 6. Admin Panel Management (`/api/v1/admin`)
| Method | Endpoint | Description | Auth |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/admin/login` | Admin panel login | Public |
| `GET` | `/api/v1/admin/dashboard` | Real-time analytics (revenue, tickets, breakdown) | Admin |
| `POST` | `/api/v1/admin/upload` | Upload event poster image (multipart form data) | Admin |
| `POST` | `/api/v1/admin/events` | Create new concert event and ticket categories | Admin |
| `PUT` | `/api/v1/admin/events/:id` | Update concert event details | Admin |
| `PATCH` | `/api/v1/admin/events/:id/toggle-close` | Open or close concert ticket sales | Admin |
| `DELETE` | `/api/v1/admin/events/:id` | Delete concert event and categories | Admin |
| `POST` | `/api/v1/admin/events/:id/categories` | Add ticket category to event | Admin |
| `PUT` | `/api/v1/admin/categories/:id` | Update ticket category (price, name, quota) | Admin |
| `DELETE` | `/api/v1/admin/categories/:id` | Delete ticket category | Admin |
| `GET` | `/api/v1/admin/orders` | List all transactions (`?search=` and `?status=`) | Admin |
| `PATCH` | `/api/v1/admin/orders/:id/status` | Update transaction status (`CHECKED_IN`, `REMINDED`, etc.) | Admin |
| `GET` | `/api/v1/admin/refunds` | List all refund submissions | Admin |
| `PATCH` | `/api/v1/admin/refunds/:id/status` | Approve or reject refund with auto quota restock | Admin |

---

## Detailed Module Documentation

### Module 1: Events and Concerts (Public)

#### 1. List All Concert Events
- **Method / URL**: `GET /api/v1/events`
- **Auth**: None
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Berhasil mengambil data konser",
  "data": [
    {
      "id": "evt-1",
      "title": "Symphony No. 5 in C minor",
      "subtitle": "Opus 67: I. Allegro con brio Masterpiece",
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
      "description": "Ludwig van Beethoven orchestral masterpiece...",
      "isClosed": false,
      "rundown": [
        { "time": "18:00 WIB", "activity": "Open Gate & Ticket Registration" },
        { "time": "19:30 WIB", "activity": "Main Performance Commences" }
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

#### 2. Get Concert Event Details
- **Method / URL**: `GET /api/v1/events/:id` (e.g. `/api/v1/events/evt-1`)
- **Auth**: None
- **Response (200 OK)**: Returns single `EventItem` object.
- **Response (404 Not Found)**:
```json
{
  "success": false,
  "message": "Konser tidak ditemukan"
}
```

---

### Module 2: User Authentication (`/api/v1/auth`)

#### 1. Request Registration OTP
- **Method / URL**: `POST /api/v1/auth/register/request-otp`
- **Body**:
```json
{
  "email": "user@example.com",
  "name": "John Doe"
}
```
- **Response (200 OK)**: OTP sent to user email (expires in 5 minutes).

#### 2. Verify Registration OTP and Create Account
- **Method / URL**: `POST /api/v1/auth/register/verify-otp`
- **Body**:
```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "otpCode": "123456",
  "password": "SecurePassword123!"
}
```
- **Response (201 Created)**: Returns JWT auth `token` and `user` profile object.

#### 3. Password Login
- **Method / URL**: `POST /api/v1/auth/login/password`
- **Body**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```
- **Response (200 OK)**: Returns JWT auth `token` and `user` profile object.

#### 4. Passwordless Login via OTP
1. **Request Login OTP**: `POST /api/v1/auth/login/request-otp`
   ```json
   { "email": "user@example.com" }
   ```
2. **Verify Login OTP**: `POST /api/v1/auth/login/verify-otp`
   ```json
   { "email": "user@example.com", "otpCode": "654321" }
   ```
   *Response (200 OK)*: Returns JWT `token` and `user` profile object.

#### 5. Password Reset Workflow
1. **Request Reset OTP**: `POST /api/v1/auth/forgot-password/request-otp`
   ```json
   { "email": "user@example.com" }
   ```
2. **Verify Reset OTP**: `POST /api/v1/auth/forgot-password/verify-otp`
   ```json
   { "email": "user@example.com", "otpCode": "112233" }
   ```
   *Response (200 OK)*: Returns `resetToken`.
3. **Set New Password**: `POST /api/v1/auth/forgot-password/reset`
   ```json
   {
     "resetToken": "eyJhbGciOiJIUzI1Ni...",
     "newPassword": "NewSecurePassword123!"
   }
   ```

#### 6. Get Current User Profile
- **Method / URL**: `GET /api/v1/auth/me`
- **Header**: `Authorization: Bearer <token>`
- **Response (200 OK)**: Returns authenticated `UserRecord`.

---

### Module 3: Customer Account Management (`/api/v1/user`)

All routes in this module require: `Authorization: Bearer <token>`

#### 1. Update Profile
- **Method / URL**: `PUT /api/v1/user/profile`
- **Body**:
```json
{
  "name": "John Doe Updated",
  "phone": "081234567890"
}
```

#### 2. Change Password
- **Method / URL**: `POST /api/v1/user/change-password`
- **Body**:
```json
{
  "oldPassword": "CurrentPassword123!",
  "newPassword": "NewStrongPassword123!"
}
```

#### 3. User Dashboard Summary
- **Method / URL**: `GET /api/v1/user/dashboard-summary`
- **Response (200 OK)**:
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

#### 4. User Order History
- **Method / URL**: `GET /api/v1/user/orders`
- **Query Filters**: `?status=ISSUED`, `?status=CHECKED_IN`, or `?status=REFUNDED`

#### 5. User Specific Order
- **Method / URL**: `GET /api/v1/user/orders/:orderCode` (e.g. `/api/v1/user/orders/SYM-849201`)

#### 6. User Refunds List
- **Method / URL**: `GET /api/v1/user/refunds`

---

### Module 4: Ticket Orders and Verification

#### 1. Checkout Ticket Order (`POST /api/v1/orders`)
Handles ticket reservation with atomic row locking (`SELECT ... FOR UPDATE`) to prevent overselling during high concurrency.

- **Authenticated Checkout**: When `Authorization: Bearer <token>` is present, `userName` and `userEmail` are inferred automatically.
- **Guest Checkout**: Supply `userName` and `userEmail` explicitly.

- **Request Body (Authenticated)**:
```json
{
  "eventId": "evt-1",
  "ticketCategoryId": "c1-vip",
  "quantity": 2
}
```

- **Request Body (Guest)**:
```json
{
  "eventId": "evt-1",
  "ticketCategoryId": "c1-vip",
  "quantity": 2,
  "userName": "Jane Doe",
  "userEmail": "jane@example.com"
}
```

- **Response (201 Created)**:
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
    "userName": "Jane Doe",
    "userEmail": "jane@example.com",
    "qrCode": "QR-SYM-849201",
    "status": "VERIFIED",
    "paymentMethod": "SANDBOX_PAYMENT",
    "createdAt": "2026-08-04T07:15:00Z"
  }
}
```

#### 2. Public Ticket Lookup
- **Method / URL**: `GET /api/v1/tickets/lookup?code=SYM-849201`
- **Auth**: None
- **Response (200 OK)**: Returns verified order record for pass printing and scanning.

---

### Module 5: Public Refunds (`/api/v1/refunds`)

#### 1. Request Refund OTP
- **Method / URL**: `POST /api/v1/refunds/request-otp`
- **Body**:
```json
{
  "orderCode": "SYM-849201",
  "userEmail": "jane@example.com"
}
```
- **Rules**: Email must match ticket holder. Orders already marked `CHECKED_IN`, `REFUNDED`, or `CANCELLED` cannot be refunded.

#### 2. Submit Refund Application
- **Method / URL**: `POST /api/v1/refunds/submit`
- **Body**:
```json
{
  "orderCode": "SYM-849201",
  "userEmail": "jane@example.com",
  "otpCode": "847291",
  "bankName": "Bank Central Asia (BCA)",
  "accountNumber": "1234567890",
  "accountHolder": "Jane Doe",
  "reason": "Sudden business travel"
}
```

#### 3. Check Refund Status
- **Method / URL**: `POST /api/v1/refunds/status`
- **Body**:
```json
{
  "orderCode": "SYM-849201",
  "userEmail": "jane@example.com"
}
```

---

### Module 6: Admin Management (`/api/v1/admin`)

#### 1. Admin Login
- **Method / URL**: `POST /api/v1/admin/login`
- **Body**: `{ "username": "admin", "password": "123" }`
- **Response (200 OK)**: Returns session admin token.

#### 2. Admin Dashboard Metrics
- **Method / URL**: `GET /api/v1/admin/dashboard`
- **Response (200 OK)**: Returns revenue, sold tickets, remaining capacity, event breakdowns, and recent orders.

#### 3. Upload Event Poster Image
- **Method / URL**: `POST /api/v1/admin/upload`
- **Content-Type**: `multipart/form-data` (Field: `image`)

#### 4. Event CRUD
- `POST /api/v1/admin/events`: Create concert event with initial categories.
- `PUT /api/v1/admin/events/:id`: Update concert details.
- `PATCH /api/v1/admin/events/:id/toggle-close`: Close or reopen ticket sales.
- `DELETE /api/v1/admin/events/:id`: Delete event and cascade ticket categories.

#### 5. Category CRUD
- `POST /api/v1/admin/events/:id/categories`: Add category to event.
- `PUT /api/v1/admin/categories/:id`: Update category price, name, and quota.
- `DELETE /api/v1/admin/categories/:id`: Remove category.

#### 6. Order Management
- `GET /api/v1/admin/orders`: List transactions with optional search (`?search=`) and status filter (`?status=`).
- `PATCH /api/v1/admin/orders/:id/status`: Update status (`CHECKED_IN`, `REMINDED`, `CANCELLED`). Updating to `REMINDED` triggers automated reminder email.

#### 7. Refund Review
- `GET /api/v1/admin/refunds`: List pending and past refund submissions.
- `PATCH /api/v1/admin/refunds/:id/status`: Approve (`APPROVED`) or reject (`REJECTED`) refund.
  - When approved, order status transitions to `REFUNDED` and category `remaining_quota` is automatically restored.

---

## Security & OTP Rate Limiting

- **Expiration**: OTP expires after 5 minutes for authentication and 10 minutes for refunds.
- **Cooldown**: Resend cooldown of 60 seconds enforced. Requests within the window return HTTP 429.
- **Max Attempts**: Maximum 5 invalid attempts allowed before OTP is revoked.

---

## Database Schema (PostgreSQL)

- `events`: Master concert records, venues, dates, conductor, audio samples, and JSONB rundowns.
- `ticket_categories`: Seating tiers (VIP, Reguler), price, total quota, and available quota per event.
- `orders`: Purchase transactions, status (`ISSUED`, `VERIFIED`, `CHECKED_IN`, `REFUNDED`), price, and QR code string.
- `refund_requests`: Customer refund applications, bank details, reason, status, and OTP verification records.
- `users`: User accounts with bcrypt password hashes, profile details, and role (`USER` or `ADMIN`).
- `auth_otps`: OTP dispatch and validation log for registration, login, and password resets.

---

## Email Testing with Mailpit

In local development, all outbound transactional email is sent to Mailpit:
- Open browser at `http://localhost:8025`.
- Inspect messages for:
  - Account OTP codes (Registration, Login, Password Reset)
  - E-Tickets with QR codes and venue guides
  - Concert reminder notifications
  - Refund approval or rejection notifications
