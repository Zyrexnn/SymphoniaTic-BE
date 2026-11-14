package repositories

import (
	"context"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

var (
	_ EventRepository  = (*MockEventRepository)(nil)
	_ OrderRepository  = (*MockOrderRepository)(nil)
	_ RefundRepository = (*MockRefundRepository)(nil)
	_ UserRepository   = (*MockUserRepository)(nil)
)

type MockEventRepository struct {
	GetAllEventsFunc            func(ctx context.Context) ([]models.EventItem, error)
	GetEventByIDFunc            func(ctx context.Context, id string) (*models.EventItem, error)
	CreateEventFunc             func(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error)
	UpdateEventFunc             func(ctx context.Context, id string, req *models.UpdateEventRequest) error
	DeleteEventFunc             func(ctx context.Context, id string) error
	ToggleEventCloseFunc        func(ctx context.Context, id string) (bool, error)
	AddCategoryFunc             func(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error)
	UpdateCategoryFunc          func(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error
	DeleteCategoryFunc          func(ctx context.Context, catID string) error
	GetCategoriesByEventIDsFunc func(ctx context.Context, eventIDs []string) (map[string][]models.TicketCategory, error)
	GetCategoriesByEventIDFunc  func(ctx context.Context, eventID string) ([]models.TicketCategory, error)
}

func (m *MockEventRepository) GetAllEvents(ctx context.Context) ([]models.EventItem, error) {
	if m.GetAllEventsFunc != nil {
		return m.GetAllEventsFunc(ctx)
	}
	return nil, nil
}

func (m *MockEventRepository) GetEventByID(ctx context.Context, id string) (*models.EventItem, error) {
	if m.GetEventByIDFunc != nil {
		return m.GetEventByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockEventRepository) CreateEvent(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error) {
	if m.CreateEventFunc != nil {
		return m.CreateEventFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockEventRepository) UpdateEvent(ctx context.Context, id string, req *models.UpdateEventRequest) error {
	if m.UpdateEventFunc != nil {
		return m.UpdateEventFunc(ctx, id, req)
	}
	return nil
}

func (m *MockEventRepository) DeleteEvent(ctx context.Context, id string) error {
	if m.DeleteEventFunc != nil {
		return m.DeleteEventFunc(ctx, id)
	}
	return nil
}

func (m *MockEventRepository) ToggleEventClose(ctx context.Context, id string) (bool, error) {
	if m.ToggleEventCloseFunc != nil {
		return m.ToggleEventCloseFunc(ctx, id)
	}
	return false, nil
}

func (m *MockEventRepository) AddCategory(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error) {
	if m.AddCategoryFunc != nil {
		return m.AddCategoryFunc(ctx, eventID, req)
	}
	return nil, nil
}

func (m *MockEventRepository) UpdateCategory(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error {
	if m.UpdateCategoryFunc != nil {
		return m.UpdateCategoryFunc(ctx, catID, req)
	}
	return nil
}

func (m *MockEventRepository) DeleteCategory(ctx context.Context, catID string) error {
	if m.DeleteCategoryFunc != nil {
		return m.DeleteCategoryFunc(ctx, catID)
	}
	return nil
}

func (m *MockEventRepository) GetCategoriesByEventIDs(ctx context.Context, eventIDs []string) (map[string][]models.TicketCategory, error) {
	if m.GetCategoriesByEventIDsFunc != nil {
		return m.GetCategoriesByEventIDsFunc(ctx, eventIDs)
	}
	return make(map[string][]models.TicketCategory), nil
}

func (m *MockEventRepository) GetCategoriesByEventID(ctx context.Context, eventID string) ([]models.TicketCategory, error) {
	if m.GetCategoriesByEventIDFunc != nil {
		return m.GetCategoriesByEventIDFunc(ctx, eventID)
	}
	return nil, nil
}

type MockOrderRepository struct {
	CreateOrderFunc              func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error)
	GetOrderByCodeFunc           func(ctx context.Context, code string) (*models.OrderRecord, error)
	GetOrderByIDFunc             func(ctx context.Context, id string) (*models.OrderRecord, error)
	LookupTicketByCodeFunc       func(ctx context.Context, code string) (*models.OrderRecord, error)
	CheckInTicketFunc            func(ctx context.Context, orderCode string) (*models.OrderRecord, error)
	GetUserOrdersFunc            func(ctx context.Context, userID, userEmail, statusFilter string) ([]models.OrderRecord, error)
	GetUserOrderByCodeFunc       func(ctx context.Context, code, userID, userEmail string) (*models.OrderRecord, error)
	GetAllOrdersFunc             func(ctx context.Context, searchQuery, statusFilter string) ([]models.OrderRecord, error)
	UpdateOrderStatusFunc        func(ctx context.Context, id, status string) error
	GetAdminDashboardMetricsFunc func(ctx context.Context) (*models.AdminDashboardMetrics, error)
	GetUserDashboardSummaryFunc  func(ctx context.Context, userID, userEmail string) (*models.UserDashboardSummary, error)
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error) {
	if m.CreateOrderFunc != nil {
		return m.CreateOrderFunc(ctx, req, authenticatedUserID)
	}
	return nil, nil
}

func (m *MockOrderRepository) GetOrderByCode(ctx context.Context, code string) (*models.OrderRecord, error) {
	if m.GetOrderByCodeFunc != nil {
		return m.GetOrderByCodeFunc(ctx, code)
	}
	return nil, nil
}

func (m *MockOrderRepository) GetOrderByID(ctx context.Context, id string) (*models.OrderRecord, error) {
	if m.GetOrderByIDFunc != nil {
		return m.GetOrderByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockOrderRepository) LookupTicketByCode(ctx context.Context, code string) (*models.OrderRecord, error) {
	if m.LookupTicketByCodeFunc != nil {
		return m.LookupTicketByCodeFunc(ctx, code)
	}
	return nil, nil
}

func (m *MockOrderRepository) CheckInTicket(ctx context.Context, orderCode string) (*models.OrderRecord, error) {
	if m.CheckInTicketFunc != nil {
		return m.CheckInTicketFunc(ctx, orderCode)
	}
	return nil, nil
}

func (m *MockOrderRepository) GetUserOrders(ctx context.Context, userID, userEmail, statusFilter string) ([]models.OrderRecord, error) {
	if m.GetUserOrdersFunc != nil {
		return m.GetUserOrdersFunc(ctx, userID, userEmail, statusFilter)
	}
	return nil, nil
}

func (m *MockOrderRepository) GetUserOrderByCode(ctx context.Context, code, userID, userEmail string) (*models.OrderRecord, error) {
	if m.GetUserOrderByCodeFunc != nil {
		return m.GetUserOrderByCodeFunc(ctx, code, userID, userEmail)
	}
	return nil, nil
}

func (m *MockOrderRepository) GetAllOrders(ctx context.Context, searchQuery, statusFilter string) ([]models.OrderRecord, error) {
	if m.GetAllOrdersFunc != nil {
		return m.GetAllOrdersFunc(ctx, searchQuery, statusFilter)
	}
	return nil, nil
}

func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, id, status string) error {
	if m.UpdateOrderStatusFunc != nil {
		return m.UpdateOrderStatusFunc(ctx, id, status)
	}
	return nil
}

func (m *MockOrderRepository) GetAdminDashboardMetrics(ctx context.Context) (*models.AdminDashboardMetrics, error) {
	if m.GetAdminDashboardMetricsFunc != nil {
		return m.GetAdminDashboardMetricsFunc(ctx)
	}
	return nil, nil
}

func (m *MockOrderRepository) GetUserDashboardSummary(ctx context.Context, userID, userEmail string) (*models.UserDashboardSummary, error) {
	if m.GetUserDashboardSummaryFunc != nil {
		return m.GetUserDashboardSummaryFunc(ctx, userID, userEmail)
	}
	return nil, nil
}

type MockRefundRepository struct {
	RequestRefundOTPFunc   func(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error
	GetRefundByOrderCodeFunc func(ctx context.Context, orderCode string) (*models.RefundRequestRecord, error)
	GetRefundWithOTPFunc   func(ctx context.Context, orderCode string) (*models.RefundRequestRecord, string, time.Time, error)
	GetRefundStatusFunc    func(ctx context.Context, orderCode, userEmail string) (*models.RefundRequestRecord, string, error)
	SubmitRefundFunc       func(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error)
	GetAllRefundsFunc      func(ctx context.Context) ([]models.RefundRequestRecord, error)
	GetUserRefundsFunc     func(ctx context.Context, userEmail string) ([]models.RefundRequestRecord, error)
	UpdateRefundStatusFunc func(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error)
}

func (m *MockRefundRepository) RequestRefundOTP(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error {
	if m.RequestRefundOTPFunc != nil {
		return m.RequestRefundOTPFunc(ctx, orderCode, userEmail, otpCode, expiresAt, refundAmount, orderID)
	}
	return nil
}

func (m *MockRefundRepository) GetRefundByOrderCode(ctx context.Context, orderCode string) (*models.RefundRequestRecord, error) {
	if m.GetRefundByOrderCodeFunc != nil {
		return m.GetRefundByOrderCodeFunc(ctx, orderCode)
	}
	return nil, nil
}

func (m *MockRefundRepository) GetRefundWithOTP(ctx context.Context, orderCode string) (*models.RefundRequestRecord, string, time.Time, error) {
	if m.GetRefundWithOTPFunc != nil {
		return m.GetRefundWithOTPFunc(ctx, orderCode)
	}
	return nil, "", time.Time{}, nil
}

func (m *MockRefundRepository) GetRefundStatus(ctx context.Context, orderCode, userEmail string) (*models.RefundRequestRecord, string, error) {
	if m.GetRefundStatusFunc != nil {
		return m.GetRefundStatusFunc(ctx, orderCode, userEmail)
	}
	return nil, "", nil
}

func (m *MockRefundRepository) SubmitRefund(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error) {
	if m.SubmitRefundFunc != nil {
		return m.SubmitRefundFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockRefundRepository) GetAllRefunds(ctx context.Context) ([]models.RefundRequestRecord, error) {
	if m.GetAllRefundsFunc != nil {
		return m.GetAllRefundsFunc(ctx)
	}
	return nil, nil
}

func (m *MockRefundRepository) GetUserRefunds(ctx context.Context, userEmail string) ([]models.RefundRequestRecord, error) {
	if m.GetUserRefundsFunc != nil {
		return m.GetUserRefundsFunc(ctx, userEmail)
	}
	return nil, nil
}

func (m *MockRefundRepository) UpdateRefundStatus(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error) {
	if m.UpdateRefundStatusFunc != nil {
		return m.UpdateRefundStatusFunc(ctx, refundID, newStatus, adminNote)
	}
	return nil, nil
}

type MockUserRepository struct {
	GetUserByIDFunc           func(ctx context.Context, id string) (*models.UserRecord, error)
	GetUserByEmailFunc          func(ctx context.Context, email string) (*models.UserRecord, error)
	GetUserWithPasswordFunc   func(ctx context.Context, email string) (*models.UserRecord, string, error)
	GetPasswordHashFunc       func(ctx context.Context, id string) (string, error)
	CreateUserFunc            func(ctx context.Context, user *models.UserRecord, passwordHash string) error
	UpdateProfileFunc         func(ctx context.Context, id string, name, phone string) (*models.UserRecord, error)
	UpdatePasswordFunc        func(ctx context.Context, id string, newPasswordHash string) error
	UpdatePasswordByEmailFunc func(ctx context.Context, email string, newPasswordHash string) error
	CheckCooldownFunc         func(ctx context.Context, email, purpose string, cooldown time.Duration) (bool, error)
	IssueOTPFunc              func(ctx context.Context, email, purpose, otpCode string, expiresAt time.Time, cooldown time.Duration) error
	VerifyOTPFunc             func(ctx context.Context, email, purpose, otpCode string, maxAttempts int) error
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id string) (*models.UserRecord, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*models.UserRecord, error) {
	if m.GetUserByEmailFunc != nil {
		return m.GetUserByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) GetUserWithPassword(ctx context.Context, email string) (*models.UserRecord, string, error) {
	if m.GetUserWithPasswordFunc != nil {
		return m.GetUserWithPasswordFunc(ctx, email)
	}
	return nil, "", nil
}

func (m *MockUserRepository) GetPasswordHash(ctx context.Context, id string) (string, error) {
	if m.GetPasswordHashFunc != nil {
		return m.GetPasswordHashFunc(ctx, id)
	}
	return "", nil
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *models.UserRecord, passwordHash string) error {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, user, passwordHash)
	}
	return nil
}

func (m *MockUserRepository) UpdateProfile(ctx context.Context, id string, name, phone string) (*models.UserRecord, error) {
	if m.UpdateProfileFunc != nil {
		return m.UpdateProfileFunc(ctx, id, name, phone)
	}
	return nil, nil
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id string, newPasswordHash string) error {
	if m.UpdatePasswordFunc != nil {
		return m.UpdatePasswordFunc(ctx, id, newPasswordHash)
	}
	return nil
}

func (m *MockUserRepository) UpdatePasswordByEmail(ctx context.Context, email string, newPasswordHash string) error {
	if m.UpdatePasswordByEmailFunc != nil {
		return m.UpdatePasswordByEmailFunc(ctx, email, newPasswordHash)
	}
	return nil
}

func (m *MockUserRepository) CheckCooldown(ctx context.Context, email, purpose string, cooldown time.Duration) (bool, error) {
	if m.CheckCooldownFunc != nil {
		return m.CheckCooldownFunc(ctx, email, purpose, cooldown)
	}
	return false, nil
}

func (m *MockUserRepository) IssueOTP(ctx context.Context, email, purpose, otpCode string, expiresAt time.Time, cooldown time.Duration) error {
	if m.IssueOTPFunc != nil {
		return m.IssueOTPFunc(ctx, email, purpose, otpCode, expiresAt, cooldown)
	}
	return nil
}

func (m *MockUserRepository) VerifyOTP(ctx context.Context, email, purpose, otpCode string, maxAttempts int) error {
	if m.VerifyOTPFunc != nil {
		return m.VerifyOTPFunc(ctx, email, purpose, otpCode, maxAttempts)
	}
	return nil
}
