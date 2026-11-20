package repositories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

func TestInterfaceConformance(t *testing.T) {
	var _ EventRepository = (*eventRepository)(nil)
	var _ OrderRepository = (*orderRepository)(nil)
	var _ RefundRepository = (*refundRepository)(nil)
	var _ UserRepository = (*userRepository)(nil)

	var _ EventRepository = (*MockEventRepository)(nil)
	var _ OrderRepository = (*MockOrderRepository)(nil)
	var _ RefundRepository = (*MockRefundRepository)(nil)
	var _ UserRepository = (*MockUserRepository)(nil)
}

func TestMockEventRepository(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockEventRepository{}

	// Default behavior (nil functions return zero values)
	events, err := mockRepo.GetAllEvents(ctx)
	if err != nil || events != nil {
		t.Fatalf("ekspektasi nilai awal nil, didapat err: %v, events: %v", err, events)
	}

	event, err := mockRepo.GetEventByID(ctx, "evt-1")
	if err != nil || event != nil {
		t.Fatalf("ekspektasi nilai awal nil, didapat err: %v, event: %v", err, event)
	}

	// Mocking behavior
	mockRepo.GetAllEventsFunc = func(ctx context.Context) ([]models.EventItem, error) {
		return []models.EventItem{
			{ID: "evt-1", Title: "Simfoni Mahakarya"},
		}, nil
	}

	events, err = mockRepo.GetAllEvents(ctx)
	if err != nil || len(events) != 1 || events[0].ID != "evt-1" {
		t.Fatalf("gagal mengeksekusi mock GetAllEventsFunc")
	}

	mockRepo.GetEventByIDFunc = func(ctx context.Context, id string) (*models.EventItem, error) {
		if id == "evt-1" {
			return &models.EventItem{ID: "evt-1", Title: "Simfoni Mahakarya"}, nil
		}
		return nil, ErrNotFound
	}

	event, err = mockRepo.GetEventByID(ctx, "evt-1")
	if err != nil || event.Title != "Simfoni Mahakarya" {
		t.Fatalf("gagal mengambil konser dari mock: %v", err)
	}

	_, err = mockRepo.GetEventByID(ctx, "evt-tidak-ada")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ekspektasi ErrNotFound, didapat: %v", err)
	}

	mockRepo.ToggleEventCloseFunc = func(ctx context.Context, id string) (bool, error) {
		return true, nil
	}
	closed, err := mockRepo.ToggleEventClose(ctx, "evt-1")
	if err != nil || !closed {
		t.Fatalf("gagal mengeksekusi mock ToggleEventCloseFunc")
	}

	mockRepo.AddCategoryFunc = func(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error) {
		return &models.TicketCategory{ID: "cat-1", EventID: eventID, Name: req.Name, Price: req.Price}, nil
	}
	cat, err := mockRepo.AddCategory(ctx, "evt-1", &models.CreateCategoryInput{Name: "VIP", Price: 500000})
	if err != nil || cat.ID != "cat-1" {
		t.Fatalf("gagal mengeksekusi mock AddCategoryFunc")
	}

	mockRepo.UpdateCategoryFunc = func(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error {
		return nil
	}
	if err := mockRepo.UpdateCategory(ctx, "cat-1", &models.UpdateTicketCategoryRequest{Name: "VIP Updated"}); err != nil {
		t.Fatalf("gagal mengeksekusi mock UpdateCategoryFunc")
	}

	mockRepo.DeleteCategoryFunc = func(ctx context.Context, catID string) error {
		return nil
	}
	if err := mockRepo.DeleteCategory(ctx, "cat-1"); err != nil {
		t.Fatalf("gagal mengeksekusi mock DeleteCategoryFunc")
	}

	mockRepo.GetCategoriesByEventIDsFunc = func(ctx context.Context, eventIDs []string) (map[string][]models.TicketCategory, error) {
		return map[string][]models.TicketCategory{
			"evt-1": {{ID: "cat-1", EventID: "evt-1", Name: "VIP"}},
		}, nil
	}
	catMap, err := mockRepo.GetCategoriesByEventIDs(ctx, []string{"evt-1"})
	if err != nil || len(catMap["evt-1"]) != 1 {
		t.Fatalf("gagal mengeksekusi mock GetCategoriesByEventIDsFunc")
	}
}

func TestMockOrderRepository(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockOrderRepository{}

	mockRepo.CreateOrderFunc = func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
		if req.Quantity > 4 {
			return nil, ErrQuotaExceeded
		}
		return &models.OrderRecord{
			ID:        "ord-1",
			OrderCode: "SYM-123456",
			UserID:    authenticatedUserID,
			Quantity:  req.Quantity,
			Status:    "ISSUED",
		}, nil
	}

	order, err := mockRepo.CreateOrder(ctx, &models.CreateOrderRequest{Quantity: 2}, "usr-1")
	if err != nil || order.OrderCode != "SYM-123456" {
		t.Fatalf("gagal mengeksekusi mock CreateOrderFunc: %v", err)
	}

	_, err = mockRepo.CreateOrder(ctx, &models.CreateOrderRequest{Quantity: 10}, "usr-1")
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("ekspektasi ErrQuotaExceeded, didapat: %v", err)
	}

	mockRepo.GetAdminDashboardMetricsFunc = func(ctx context.Context) (*models.AdminDashboardMetrics, error) {
		return &models.AdminDashboardMetrics{
			TotalRevenue: 5000000,
			TicketsSold:  10,
			TotalOrders:  5,
		}, nil
	}

	metrics, err := mockRepo.GetAdminDashboardMetrics(ctx)
	if err != nil || metrics.TotalRevenue != 5000000 {
		t.Fatalf("gagal mengeksekusi mock GetAdminDashboardMetricsFunc")
	}

	mockRepo.GetUserDashboardSummaryFunc = func(ctx context.Context, userID, userEmail string) (*models.UserDashboardSummary, error) {
		return &models.UserDashboardSummary{
			TotalTicketsBought:  10,
			UpcomingEventsCount: 2,
		}, nil
	}

	summary, err := mockRepo.GetUserDashboardSummary(ctx, "usr-1", "user@test.com")
	if err != nil || summary.TotalTicketsBought != 10 {
		t.Fatalf("gagal mengeksekusi mock GetUserDashboardSummaryFunc")
	}

	mockRepo.CheckInTicketFunc = func(ctx context.Context, orderCode string) (*models.OrderRecord, error) {
		if orderCode == "SYM-USED" {
			return nil, ErrTicketAlreadyUsed
		}
		return &models.OrderRecord{
			OrderCode: orderCode,
			Status:    "CHECKED_IN",
		}, nil
	}
	checkedInOrder, err := mockRepo.CheckInTicket(ctx, "SYM-123456")
	if err != nil || checkedInOrder.Status != "CHECKED_IN" {
		t.Fatalf("gagal mengeksekusi mock CheckInTicketFunc: %v", err)
	}
	_, err = mockRepo.CheckInTicket(ctx, "SYM-USED")
	if !errors.Is(err, ErrTicketAlreadyUsed) {
		t.Fatalf("ekspektasi ErrTicketAlreadyUsed, didapat: %v", err)
	}
}

func TestMockRefundRepository(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockRefundRepository{}

	mockRepo.RequestRefundOTPFunc = func(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error {
		return nil
	}
	if err := mockRepo.RequestRefundOTP(ctx, "SYM-123", "test@test.com", "123456", time.Now().Add(10*time.Minute), 500000, "ord-1"); err != nil {
		t.Fatalf("gagal mengeksekusi mock RequestRefundOTPFunc")
	}

	mockRepo.SubmitRefundFunc = func(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error) {
		if input.OTPCode != "123456" {
			return nil, ErrOTPInvalid
		}
		return &models.RefundRequestRecord{
			OrderCode: input.OrderCode,
			Status:    "PENDING",
		}, nil
	}

	refund, err := mockRepo.SubmitRefund(ctx, &models.SubmitRefundInput{OrderCode: "SYM-123", OTPCode: "123456"})
	if err != nil || refund.Status != "PENDING" {
		t.Fatalf("gagal mengeksekusi mock SubmitRefundFunc: %v", err)
	}

	_, err = mockRepo.SubmitRefund(ctx, &models.SubmitRefundInput{OrderCode: "SYM-123", OTPCode: "999999"})
	if !errors.Is(err, ErrOTPInvalid) {
		t.Fatalf("ekspektasi ErrOTPInvalid, didapat: %v", err)
	}

	mockRepo.UpdateRefundStatusFunc = func(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error) {
		return &models.RefundRequestRecord{
			ID:        refundID,
			Status:    newStatus,
			AdminNote: adminNote,
		}, nil
	}

	updated, err := mockRepo.UpdateRefundStatus(ctx, "rf-1", "APPROVED", "Disetujui")
	if err != nil || updated.Status != "APPROVED" {
		t.Fatalf("gagal mengeksekusi mock UpdateRefundStatusFunc")
	}
}

func TestMockUserRepository(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockUserRepository{}

	mockRepo.GetUserByEmailFunc = func(ctx context.Context, email string) (*models.UserRecord, error) {
		if email == "exist@test.com" {
			return &models.UserRecord{ID: "usr-1", Email: email, Name: "Pengguna Uji"}, nil
		}
		return nil, ErrNotFound
	}

	user, err := mockRepo.GetUserByEmail(ctx, "exist@test.com")
	if err != nil || user.ID != "usr-1" {
		t.Fatalf("gagal mengeksekusi mock GetUserByEmailFunc")
	}

	_, err = mockRepo.GetUserByEmail(ctx, "unknown@test.com")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ekspektasi ErrNotFound, didapat: %v", err)
	}

	mockRepo.CheckCooldownFunc = func(ctx context.Context, email, purpose string, cooldown time.Duration) (bool, error) {
		if email == "cooldown@test.com" {
			return true, nil
		}
		return false, nil
	}

	isCooldown, err := mockRepo.CheckCooldown(ctx, "cooldown@test.com", "LOGIN", 60*time.Second)
	if err != nil || !isCooldown {
		t.Fatalf("ekspektasi isCooldown true, didapat: %v", isCooldown)
	}

	mockRepo.VerifyOTPFunc = func(ctx context.Context, email, purpose, otpCode string, maxAttempts int) error {
		if otpCode != "654321" {
			return ErrOTPInvalid
		}
		return nil
	}

	if err := mockRepo.VerifyOTP(ctx, "user@test.com", "LOGIN", "654321", 5); err != nil {
		t.Fatalf("gagal verifikasi OTP valid: %v", err)
	}

	if err := mockRepo.VerifyOTP(ctx, "user@test.com", "LOGIN", "000000", 5); !errors.Is(err, ErrOTPInvalid) {
		t.Fatalf("ekspektasi ErrOTPInvalid, didapat: %v", err)
	}
}

func TestNewConstructors(t *testing.T) {
	// Memastikan fungsi konstruktor mengembalikan implementasi konkret yang memenuhi antarmuka
	eventRepo := NewEventRepository(nil)
	if eventRepo == nil {
		t.Fatal("NewEventRepository mengembalikan nil")
	}

	orderRepo := NewOrderRepository(nil)
	if orderRepo == nil {
		t.Fatal("NewOrderRepository mengembalikan nil")
	}

	refundRepo := NewRefundRepository(nil)
	if refundRepo == nil {
		t.Fatal("NewRefundRepository mengembalikan nil")
	}

	userRepo := NewUserRepository(nil)
	if userRepo == nil {
		t.Fatal("NewUserRepository mengembalikan nil")
	}
}

type recordedSQLCall struct {
	query string
	args  []driver.Value
}

type stubQueryResult struct {
	columns []string
	rows    [][]driver.Value
	err     error
}

type stubExecResult struct {
	affected int64
	err      error
}

type stubDBState struct {
	mu          sync.Mutex
	calls       []recordedSQLCall
	queryRouter func(query string, args []driver.Value) stubQueryResult
	execRouter  func(query string, args []driver.Value) stubExecResult
}

type stubDriver struct {
	state *stubDBState
}

func (d *stubDriver) Open(name string) (driver.Conn, error) {
	return &stubConn{state: d.state}, nil
}

type stubConn struct {
	state *stubDBState
}

func (c *stubConn) Prepare(query string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (c *stubConn) Close() error                              { return nil }
func (c *stubConn) Begin() (driver.Tx, error)                 { return &stubTx{}, nil }
func (c *stubConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return &stubTx{}, nil
}

func (c *stubConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	vals := make([]driver.Value, len(args))
	for i, a := range args {
		vals[i] = a.Value
	}
	c.state.mu.Lock()
	c.state.calls = append(c.state.calls, recordedSQLCall{query: query, args: vals})
	router := c.state.execRouter
	c.state.mu.Unlock()

	if router != nil {
		res := router(query, vals)
		if res.err != nil {
			return nil, res.err
		}
		return driver.RowsAffected(res.affected), nil
	}
	return driver.RowsAffected(1), nil
}

func (c *stubConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	vals := make([]driver.Value, len(args))
	for i, a := range args {
		vals[i] = a.Value
	}
	c.state.mu.Lock()
	c.state.calls = append(c.state.calls, recordedSQLCall{query: query, args: vals})
	router := c.state.queryRouter
	c.state.mu.Unlock()

	if router != nil {
		res := router(query, vals)
		if res.err != nil {
			return nil, res.err
		}
		return &stubRows{columns: res.columns, rows: res.rows}, nil
	}
	return &stubRows{}, nil
}

type stubTx struct{}

func (t *stubTx) Commit() error   { return nil }
func (t *stubTx) Rollback() error { return nil }

type stubRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *stubRows) Columns() []string { return r.columns }
func (r *stubRows) Close() error      { return nil }
func (r *stubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

type stubConnector struct {
	state *stubDBState
}

func (c *stubConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return &stubConn{state: c.state}, nil
}

func (c *stubConnector) Driver() driver.Driver {
	return &stubDriver{state: c.state}
}

func newStubDB(t *testing.T, state *stubDBState) *sql.DB {
	t.Helper()
	db := sql.OpenDB(&stubConnector{state: state})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func orderStubColumns() []string {
	return []string{
		"id", "order_code", "user_id", "event_id", "event_title", "artist", "venue", "date",
		"open_gate", "event_time", "venue_address", "category_name", "quantity", "total_price",
		"user_name", "user_email", "qr_code", "status", "payment_method", "created_at",
	}
}

func sampleOrderRow(status string) []driver.Value {
	return []driver.Value{
		"ord-1", "SYM-100001", "usr-1", "evt-1", "Simfoni Siang", "Jakarta Orchestra", "Aula Simfonia",
		"22 November 2026 @ 14:21 WIB", "13:21 WIB", "14:21 WIB", "Jl. Industri No. 1",
		"VIP Pit", int64(2), float64(1500000), "Budi", "budi@example.com", "QR-SYM-100001",
		status, "SANDBOX_PAYMENT", time.Now(),
	}
}

func TestOrderRepository_EnrichedQueriesAndRemindedCheckIn(t *testing.T) {
	ctx := context.Background()
	state := &stubDBState{
		queryRouter: func(query string, args []driver.Value) stubQueryResult {
			return stubQueryResult{
				columns: orderStubColumns(),
				rows:    [][]driver.Value{sampleOrderRow("REMINDED")},
			}
		},
	}
	db := newStubDB(t, state)
	repo := NewOrderRepository(db)

	ordByCode, err := repo.GetOrderByCode(ctx, "SYM-100001")
	if err != nil {
		t.Fatalf("GetOrderByCode gagal: %v", err)
	}
	if ordByCode.OpenGate != "13:21 WIB" || ordByCode.EventTime != "14:21 WIB" || ordByCode.VenueAddress != "Jl. Industri No. 1" {
		t.Fatalf("Metadata event pada OrderRecord tidak lengkap: %+v", ordByCode)
	}

	ordByID, err := repo.GetOrderByID(ctx, "ord-1")
	if err != nil || ordByID.OpenGate != "13:21 WIB" {
		t.Fatalf("GetOrderByID gagal mengisi OpenGate: %v", err)
	}

	userOrders, err := repo.GetUserOrders(ctx, "usr-1", "budi@example.com", "")
	if err != nil || len(userOrders) != 1 || userOrders[0].VenueAddress != "Jl. Industri No. 1" {
		t.Fatalf("GetUserOrders gagal mengisi VenueAddress: %v", err)
	}

	userOrd, err := repo.GetUserOrderByCode(ctx, "SYM-100001", "usr-1", "budi@example.com")
	if err != nil || userOrd.EventTime != "14:21 WIB" {
		t.Fatalf("GetUserOrderByCode gagal mengisi EventTime: %v", err)
	}

	allOrds, err := repo.GetAllOrders(ctx, "SYM", "")
	if err != nil || len(allOrds) != 1 || allOrds[0].OpenGate != "13:21 WIB" {
		t.Fatalf("GetAllOrders gagal mengisi OpenGate: %v", err)
	}

	// Pastikan CheckInTicket menerima status REMINDED
	checkInState := &stubDBState{
		queryRouter: func(query string, args []driver.Value) stubQueryResult {
			return stubQueryResult{
				columns: []string{
					"id", "order_code", "user_id", "event_id", "event_title", "artist", "venue", "date",
					"category_name", "quantity", "total_price", "user_name", "user_email", "qr_code",
					"status", "payment_method", "created_at",
					"is_closed", "open_gate", "time", "event_date_time",
				},
				rows: [][]driver.Value{{
					"ord-1", "SYM-100001", "usr-1", "evt-1", "Simfoni Siang", "Jakarta Orchestra", "Aula Simfonia",
					time.Now().Format("2006-01-02") + " @ " + time.Now().Format("15:04"), "VIP Pit", int64(2), float64(1500000), "Budi", "budi@example.com",
					"QR-SYM-100001", "REMINDED", "SANDBOX_PAYMENT", time.Now(),
					false, "", "", time.Now().Add(-10 * time.Minute),
				}},
			}
		},
	}
	checkedIn, err := NewOrderRepository(newStubDB(t, checkInState)).CheckInTicket(ctx, "SYM-100001")
	if err != nil {
		t.Fatalf("CheckInTicket untuk tiket REMINDED seharusnya sukses, dapat: %v", err)
	}
	if checkedIn.Status != "CHECKED_IN" {
		t.Fatalf("Status setelah check-in seharusnya CHECKED_IN, dapat %s", checkedIn.Status)
	}
}

func TestOrderRepository_CheckInTicketGateValidations(t *testing.T) {
	ctx := context.Background()

	t.Run("Menolak check-in ketika konser ditutup (is_closed = true)", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: []string{
						"id", "order_code", "user_id", "event_id", "event_title", "artist", "venue", "date",
						"category_name", "quantity", "total_price", "user_name", "user_email", "qr_code",
						"status", "payment_method", "created_at",
						"is_closed", "open_gate", "time", "event_date_time",
					},
					rows: [][]driver.Value{{
						"ord-1", "SYM-CLOSED", "usr-1", "evt-1", "Simfoni Ditutup", "Jakarta Orchestra", "Aula Simfonia",
						time.Now().Format("2006-01-02") + " @ " + time.Now().Format("15:04"), "VIP Pit", int64(1), float64(750000), "Budi", "budi@example.com",
						"QR-SYM-CLOSED", "ISSUED", "SANDBOX_PAYMENT", time.Now(),
						true, "", "", time.Now().Add(-10 * time.Minute),
					}},
				}
			},
		}
		_, err := NewOrderRepository(newStubDB(t, state)).CheckInTicket(ctx, "SYM-CLOSED")
		if !errors.Is(err, ErrEventClosed) {
			t.Fatalf("Ekspektasi ErrEventClosed, dapat: %v", err)
		}
	})

	t.Run("Menolak check-in ketika konser belum dimulai / open gate belum dibuka", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: []string{
						"id", "order_code", "user_id", "event_id", "event_title", "artist", "venue", "date",
						"category_name", "quantity", "total_price", "user_name", "user_email", "qr_code",
						"status", "payment_method", "created_at",
						"is_closed", "open_gate", "time", "event_date_time",
					},
					rows: [][]driver.Value{{
						"ord-2", "SYM-FUTURE", "usr-1", "evt-1", "Simfoni Besok", "Jakarta Orchestra", "Aula Simfonia",
						futureTime.Format("2006-01-02") + " @ 19:30", "VIP Pit", int64(1), float64(750000), "Budi", "budi@example.com",
						"QR-SYM-FUTURE", "ISSUED", "SANDBOX_PAYMENT", time.Now(),
						false, "18:00", "19:30", futureTime,
					}},
				}
			},
		}
		_, err := NewOrderRepository(newStubDB(t, state)).CheckInTicket(ctx, "SYM-FUTURE")
		if !errors.Is(err, ErrEventNotStarted) {
			t.Fatalf("Ekspektasi ErrEventNotStarted, dapat: %v", err)
		}
	})

	t.Run("Menolak check-in ketika konser telah kedaluwarsa (> 6 jam setelah mulai)", func(t *testing.T) {
		pastTime := time.Now().Add(-8 * time.Hour)
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: []string{
						"id", "order_code", "user_id", "event_id", "event_title", "artist", "venue", "date",
						"category_name", "quantity", "total_price", "user_name", "user_email", "qr_code",
						"status", "payment_method", "created_at",
						"is_closed", "open_gate", "time", "event_date_time",
					},
					rows: [][]driver.Value{{
						"ord-3", "SYM-EXPIRED", "usr-1", "evt-1", "Simfoni Kemarin", "Jakarta Orchestra", "Aula Simfonia",
						pastTime.Format("2006-01-02") + " @ " + pastTime.Format("15:04"), "VIP Pit", int64(1), float64(750000), "Budi", "budi@example.com",
						"QR-SYM-EXPIRED", "ISSUED", "SANDBOX_PAYMENT", time.Now(),
						false, pastTime.Add(-1 * time.Hour).Format("15:04"), pastTime.Format("15:04"), pastTime,
					}},
				}
			},
		}
		_, err := NewOrderRepository(newStubDB(t, state)).CheckInTicket(ctx, "SYM-EXPIRED")
		if !errors.Is(err, ErrEventExpired) {
			t.Fatalf("Ekspektasi ErrEventExpired, dapat: %v", err)
		}
	})
}

func TestOrderRepository_UpdateOrderStatusQuotaRestoration(t *testing.T) {
	ctx := context.Background()

	t.Run("Mengembalikan kuota saat transisi dari status aktif ke CANCELLED", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: []string{"status", "event_id", "category_name", "quantity"},
					rows:    [][]driver.Value{{"REMINDED", "evt-1", "VIP Pit", int64(3)}},
				}
			},
		}
		repo := NewOrderRepository(newStubDB(t, state))
		if err := repo.UpdateOrderStatus(ctx, "ord-1", "CANCELLED"); err != nil {
			t.Fatalf("UpdateOrderStatus gagal: %v", err)
		}

		var hasForUpdate, hasQuotaRestore bool
		for _, c := range state.calls {
			if strings.Contains(c.query, "FOR UPDATE") && strings.Contains(c.query, "FROM orders") {
				hasForUpdate = true
			}
			if strings.Contains(c.query, "remaining_quota = remaining_quota + $1") {
				hasQuotaRestore = true
				if len(c.args) != 3 || c.args[0] != int64(3) || c.args[1] != "evt-1" || c.args[2] != "VIP Pit" {
					t.Fatalf("Argumen pengembalian kuota tidak sesuai: %+v", c.args)
				}
			}
		}
		if !hasForUpdate {
			t.Fatal("UpdateOrderStatus wajib mengunci baris orders dengan FOR UPDATE")
		}
		if !hasQuotaRestore {
			t.Fatal("UpdateOrderStatus wajib mengembalikan remaining_quota saat dibatalkan")
		}
	})

	t.Run("Tidak menggandakan kuota saat transisi dari CANCELLED ke REFUNDED", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: []string{"status", "event_id", "category_name", "quantity"},
					rows:    [][]driver.Value{{"CANCELLED", "evt-1", "VIP Pit", int64(3)}},
				}
			},
		}
		repo := NewOrderRepository(newStubDB(t, state))
		if err := repo.UpdateOrderStatus(ctx, "ord-1", "REFUNDED"); err != nil {
			t.Fatalf("UpdateOrderStatus gagal: %v", err)
		}

		for _, c := range state.calls {
			if strings.Contains(c.query, "remaining_quota") {
				t.Fatalf("Tidak boleh memulihkan kuota lagi jika status awal sudah CANCELLED")
			}
		}
	})
}

func TestOrderRepository_AdminMetricsIncludesReminded(t *testing.T) {
	ctx := context.Background()
	state := &stubDBState{
		queryRouter: func(query string, args []driver.Value) stubQueryResult {
			switch {
			case strings.Contains(query, "COALESCE(SUM(total_price), 0)"):
				return stubQueryResult{
					columns: []string{"rev", "sold", "cnt"},
					rows:    [][]driver.Value{{float64(1500000), int64(2), int64(1)}},
				}
			case strings.Contains(query, "SUM(remaining_quota)"), strings.Contains(query, "COUNT(*) FROM events"):
				return stubQueryResult{
					columns: []string{"val"},
					rows:    [][]driver.Value{{int64(10)}},
				}
			default:
				return stubQueryResult{columns: []string{"c1", "c2", "c3", "c4"}}
			}
		},
	}

	metrics, err := NewOrderRepository(newStubDB(t, state)).GetAdminDashboardMetrics(ctx)
	if err != nil {
		t.Fatalf("GetAdminDashboardMetrics gagal: %v", err)
	}
	if metrics.TotalRevenue != 1500000 || metrics.TicketsSold != 2 {
		t.Fatalf("Metrik tidak sesuai: %+v", metrics)
	}

	var statusFilteredCalls int
	for _, c := range state.calls {
		if strings.Contains(c.query, "status IN (") {
			statusFilteredCalls++
			if !strings.Contains(c.query, "'REMINDED'") {
				t.Fatalf("Query metrik admin tidak menyertakan status 'REMINDED': %s", c.query)
			}
		}
	}
	if statusFilteredCalls != 4 {
		t.Fatalf("Ekspektasi 4 query metrik memfilter status aktif termasuk REMINDED, didapat %d", statusFilteredCalls)
	}
}

func TestEventRepository_UpdateAndDeleteGuards(t *testing.T) {
	ctx := context.Background()

	t.Run("UpdateEvent menjaga audio_url dengan COALESCE(NULLIF($9, ''), audio_url)", func(t *testing.T) {
		state := &stubDBState{}
		repo := NewEventRepository(newStubDB(t, state))
		err := repo.UpdateEvent(ctx, "evt-1", &models.UpdateEventRequest{
			Title:    "Konser Siang",
			Time:     "14:21 WIB",
			OpenGate: "13:21 WIB",
			AudioURL: "",
		})
		if err != nil {
			t.Fatalf("UpdateEvent gagal: %v", err)
		}
		if len(state.calls) == 0 || !strings.Contains(state.calls[0].query, "COALESCE(NULLIF($9, ''), audio_url)") {
			t.Fatal("UpdateEvent wajib menggunakan COALESCE(NULLIF($9, ''), audio_url) agar audio lama tidak terhapus")
		}
	})

	t.Run("DeleteEvent mengembalikan ErrEventHasOrders jika masih ada transaksi tiket", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: []string{"count"},
					rows:    [][]driver.Value{{int64(2)}},
				}
			},
		}
		repo := NewEventRepository(newStubDB(t, state))
		err := repo.DeleteEvent(ctx, "evt-1")
		if !errors.Is(err, ErrEventHasOrders) {
			t.Fatalf("Ekspektasi ErrEventHasOrders, dapat: %v", err)
		}
		for _, c := range state.calls {
			if strings.Contains(c.query, "DELETE FROM events") {
				t.Fatal("DeleteEvent tidak boleh mengeksekusi DELETE jika orders masih ada")
			}
		}
	})
}

func TestRefundRepository_FiltersOTPSentDrafts(t *testing.T) {
	ctx := context.Background()
	state := &stubDBState{
		queryRouter: func(query string, args []driver.Value) stubQueryResult {
			return stubQueryResult{
				columns: []string{
					"id", "order_id", "order_code", "user_email", "bank_name", "account_number",
					"account_holder", "reason", "refund_amount", "status", "admin_note", "created_at", "updated_at",
					"event_title", "category_name", "quantity", "user_name",
				},
			}
		},
	}
	repo := NewRefundRepository(newStubDB(t, state))

	if _, err := repo.GetUserRefunds(ctx, "user@example.com"); err != nil {
		t.Fatalf("GetUserRefunds gagal: %v", err)
	}
	if _, _, err := repo.GetRefundStatus(ctx, "SYM-100001", "user@example.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Ekspektasi ErrNotFound saat hanya ada draft OTP_SENT, dapat: %v", err)
	}

	if len(state.calls) != 2 {
		t.Fatalf("Ekspektasi 2 panggilan query, didapat %d", len(state.calls))
	}
	for i, c := range state.calls {
		if !strings.Contains(c.query, "r.status != 'OTP_SENT'") {
			t.Fatalf("Query ke-%d tidak menyaring status OTP_SENT: %s", i+1, c.query)
		}
	}
}

func TestOrderRepository_GetAttendanceSummary(t *testing.T) {
	ctx := context.Background()

	t.Run("Menghitung persentase kehadiran dan sisa tiket belum masuk", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				if len(args) != 1 || args[0] != "evt-1" {
					t.Errorf("Ekspektasi argumen eventID 'evt-1', dapat %+v", args)
				}
				return stubQueryResult{
					columns: []string{"total_issued", "total_checked_in"},
					rows:    [][]driver.Value{{int64(200), int64(137)}},
				}
			},
		}
		repo := NewOrderRepository(newStubDB(t, state))
		summary, err := repo.GetAttendanceSummary(ctx, " evt-1 ")
		if err != nil {
			t.Fatalf("GetAttendanceSummary gagal: %v", err)
		}
		if summary.TotalTicketsIssued != 200 || summary.TotalCheckedIn != 137 || summary.RemainingCheckIn != 63 {
			t.Fatalf("Angka ringkasan tidak sesuai: %+v", summary)
		}
		if summary.CheckInPercentage != 68.5 {
			t.Fatalf("Ekspektasi persentase 68.5, dapat: %v", summary.CheckInPercentage)
		}
	})

	t.Run("Mengembalikan 0 persen jika belum ada tiket terbit", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: []string{"total_issued", "total_checked_in"},
					rows:    [][]driver.Value{{int64(0), int64(0)}},
				}
			},
		}
		repo := NewOrderRepository(newStubDB(t, state))
		summary, err := repo.GetAttendanceSummary(ctx, "")
		if err != nil {
			t.Fatalf("GetAttendanceSummary gagal: %v", err)
		}
		if summary.CheckInPercentage != 0 || summary.RemainingCheckIn != 0 {
			t.Fatalf("Ekspektasi 0 untuk data kosong, dapat: %+v", summary)
		}
	})
}
