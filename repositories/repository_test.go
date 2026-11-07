package repositories

import (
	"context"
	"errors"
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
