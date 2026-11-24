package repositories

import (
	"context"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

var (
	_ EventRepository     = (*MockEventRepository)(nil)
	_ OrderRepository     = (*MockOrderRepository)(nil)
	_ RefundRepository    = (*MockRefundRepository)(nil)
	_ UserRepository      = (*MockUserRepository)(nil)
	_ PromoRepository     = (*MockPromoRepository)(nil)
	_ AdminUserRepository = (*MockAdminUserRepository)(nil)
	_ BroadcastRepository = (*MockBroadcastRepository)(nil)
)

type MockEventRepository struct {
	GetAllEventsFunc            func(ctx context.Context) ([]models.EventItem, error)
	GetEventByIDFunc            func(ctx context.Context, id string) (*models.EventItem, error)
	CreateEventFunc             func(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error)
	UpdateEventFunc             func(ctx context.Context, id string, req *models.UpdateEventRequest) error
	DeleteEventFunc             func(ctx context.Context, id string) error
	ToggleEventCloseFunc        func(ctx context.Context, id string) (bool, error)
	CancelEventFunc             func(ctx context.Context, id string, reason string) (*models.EventItem, error)
	RescheduleEventFunc         func(ctx context.Context, id string, req *models.RescheduleEventRequest) (*models.EventItem, error)
	AddCategoryFunc             func(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error)
	UpdateCategoryFunc          func(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error
	DeleteCategoryFunc          func(ctx context.Context, catID string) error
	GetCategoriesByEventIDsFunc func(ctx context.Context, eventIDs []string) (map[string][]models.TicketCategory, error)
	GetCategoriesByEventIDFunc  func(ctx context.Context, eventID string) ([]models.TicketCategory, error)
	GetSeatsByEventIDFunc       func(ctx context.Context, eventID string) ([]models.SeatItem, error)
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

func (m *MockEventRepository) CancelEvent(ctx context.Context, id string, reason string) (*models.EventItem, error) {
	if m.CancelEventFunc != nil {
		return m.CancelEventFunc(ctx, id, reason)
	}
	return &models.EventItem{ID: id, Status: "CANCELLED", IsClosed: true, CancellationReason: reason}, nil
}

func (m *MockEventRepository) RescheduleEvent(ctx context.Context, id string, req *models.RescheduleEventRequest) (*models.EventItem, error) {
	if m.RescheduleEventFunc != nil {
		return m.RescheduleEventFunc(ctx, id, req)
	}
	return &models.EventItem{ID: id, Status: "RESCHEDULED", Date: req.NewDate, Time: req.NewTime, RescheduleReason: req.Reason}, nil
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

func (m *MockEventRepository) GetSeatsByEventID(ctx context.Context, eventID string) ([]models.SeatItem, error) {
	if m.GetSeatsByEventIDFunc != nil {
		return m.GetSeatsByEventIDFunc(ctx, eventID)
	}
	return nil, nil
}

type MockOrderRepository struct {
	CreateOrderFunc              func(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error)
	GetOrderByCodeFunc           func(ctx context.Context, code string) (*models.OrderRecord, error)
	GetOrderByIDFunc             func(ctx context.Context, id string) (*models.OrderRecord, error)
	LookupTicketByCodeFunc       func(ctx context.Context, code string) (*models.OrderRecord, error)
	CheckInTicketFunc            func(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error)
	GetUserOrdersFunc            func(ctx context.Context, userID, userEmail, statusFilter string) ([]models.OrderRecord, error)
	GetUserOrderByCodeFunc       func(ctx context.Context, code, userID, userEmail string) (*models.OrderRecord, error)
	GetAllOrdersFunc             func(ctx context.Context, searchQuery, statusFilter string) ([]models.OrderRecord, error)
	UpdateOrderStatusFunc        func(ctx context.Context, id, status string) error
	GetAdminDashboardMetricsFunc func(ctx context.Context) (*models.AdminDashboardMetrics, error)
	GetAttendanceSummaryFunc     func(ctx context.Context, eventID string) (*models.GateAttendanceSummary, error)
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

func (m *MockOrderRepository) CheckInTicket(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error) {
	if m.CheckInTicketFunc != nil {
		return m.CheckInTicketFunc(ctx, orderCode, bypassOpenGate)
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

func (m *MockOrderRepository) GetAttendanceSummary(ctx context.Context, eventID string) (*models.GateAttendanceSummary, error) {
	if m.GetAttendanceSummaryFunc != nil {
		return m.GetAttendanceSummaryFunc(ctx, eventID)
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
	RequestRefundOTPFunc       func(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error
	GetRefundByOrderCodeFunc   func(ctx context.Context, orderCode string) (*models.RefundRequestRecord, error)
	GetRefundWithOTPFunc       func(ctx context.Context, orderCode string) (*models.RefundRequestRecord, string, time.Time, error)
	GetRefundStatusFunc        func(ctx context.Context, orderCode, userEmail string) (*models.RefundRequestRecord, string, error)
	SubmitRefundFunc           func(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error)
	SubmitDirectUserRefundFunc func(ctx context.Context, order *models.OrderRecord, input *models.SubmitUserRefundInput) (*models.RefundRequestRecord, error)
	GetAllRefundsFunc          func(ctx context.Context) ([]models.RefundRequestRecord, error)
	GetUserRefundsFunc         func(ctx context.Context, userEmail string) ([]models.RefundRequestRecord, error)
	UpdateRefundStatusFunc     func(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error)
	BatchRefundByEventIDFunc   func(ctx context.Context, eventID string) (*models.BatchRefundResult, []models.OrderRecord, error)
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

func (m *MockRefundRepository) SubmitDirectUserRefund(ctx context.Context, order *models.OrderRecord, input *models.SubmitUserRefundInput) (*models.RefundRequestRecord, error) {
	if m.SubmitDirectUserRefundFunc != nil {
		return m.SubmitDirectUserRefundFunc(ctx, order, input)
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

func (m *MockRefundRepository) BatchRefundByEventID(ctx context.Context, eventID string) (*models.BatchRefundResult, []models.OrderRecord, error) {
	if m.BatchRefundByEventIDFunc != nil {
		return m.BatchRefundByEventIDFunc(ctx, eventID)
	}
	return &models.BatchRefundResult{EventID: eventID}, nil, nil
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

type MockPromoRepository struct {
	GetPromoByCodeFunc    func(ctx context.Context, code string) (*models.Promotion, error)
	HasUserUsedPromoFunc  func(ctx context.Context, promotionID, userID string) (bool, error)
	GetAllPromosFunc      func(ctx context.Context) ([]models.Promotion, error)
	GetActivePromosFunc   func(ctx context.Context) ([]models.Promotion, error)
	CreatePromoFunc       func(ctx context.Context, req *models.CreatePromoRequest) (*models.Promotion, error)
	UpdatePromoFunc       func(ctx context.Context, id string, req *models.UpdatePromoRequest) (*models.Promotion, error)
	TogglePromoActiveFunc func(ctx context.Context, id string) (bool, error)
	DeletePromoFunc       func(ctx context.Context, id string) error
}

func (m *MockPromoRepository) GetPromoByCode(ctx context.Context, code string) (*models.Promotion, error) {
	if m.GetPromoByCodeFunc != nil {
		return m.GetPromoByCodeFunc(ctx, code)
	}
	return nil, nil
}

func (m *MockPromoRepository) HasUserUsedPromo(ctx context.Context, promotionID, userID string) (bool, error) {
	if m.HasUserUsedPromoFunc != nil {
		return m.HasUserUsedPromoFunc(ctx, promotionID, userID)
	}
	return false, nil
}

func (m *MockPromoRepository) GetAllPromos(ctx context.Context) ([]models.Promotion, error) {
	if m.GetAllPromosFunc != nil {
		return m.GetAllPromosFunc(ctx)
	}
	return []models.Promotion{}, nil
}

func (m *MockPromoRepository) GetActivePromos(ctx context.Context) ([]models.Promotion, error) {
	if m.GetActivePromosFunc != nil {
		return m.GetActivePromosFunc(ctx)
	}
	return []models.Promotion{}, nil
}

func (m *MockPromoRepository) CreatePromo(ctx context.Context, req *models.CreatePromoRequest) (*models.Promotion, error) {
	if m.CreatePromoFunc != nil {
		return m.CreatePromoFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockPromoRepository) UpdatePromo(ctx context.Context, id string, req *models.UpdatePromoRequest) (*models.Promotion, error) {
	if m.UpdatePromoFunc != nil {
		return m.UpdatePromoFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *MockPromoRepository) TogglePromoActive(ctx context.Context, id string) (bool, error) {
	if m.TogglePromoActiveFunc != nil {
		return m.TogglePromoActiveFunc(ctx, id)
	}
	return false, nil
}

func (m *MockPromoRepository) DeletePromo(ctx context.Context, id string) error {
	if m.DeletePromoFunc != nil {
		return m.DeletePromoFunc(ctx, id)
	}
	return nil
}

type MockAdminUserRepository struct {
	GetByUsernameFunc          func(ctx context.Context, username string) (*models.AdminUser, error)
	GetByIDFunc                func(ctx context.Context, id string) (*models.AdminUser, error)
	GetAllStaffFunc            func(ctx context.Context, roleFilter string, search string) ([]models.AdminUser, error)
	CreateStaffFunc            func(ctx context.Context, req *models.CreateStaffRequest, passwordHash string) (*models.AdminUser, error)
	UpdateStaffFunc            func(ctx context.Context, id string, req *models.UpdateStaffRequest) (*models.AdminUser, error)
	ToggleStaffActiveFunc      func(ctx context.Context, id string) (*models.AdminUser, error)
	ResetStaffPasswordFunc     func(ctx context.Context, id string, passwordHash string) error
	DeleteStaffFunc            func(ctx context.Context, id string) error
	CountActiveSuperAdminsFunc func(ctx context.Context) (int, error)
	CreateAuditLogFunc         func(ctx context.Context, logEntry *models.AuditLog) error
	GetAuditLogsFunc           func(ctx context.Context, actionFilter, search string, limit int) ([]models.AuditLog, error)
}

func (m *MockAdminUserRepository) GetByUsername(ctx context.Context, username string) (*models.AdminUser, error) {
	if m.GetByUsernameFunc != nil {
		return m.GetByUsernameFunc(ctx, username)
	}
	return nil, ErrNotFound
}

func (m *MockAdminUserRepository) GetByID(ctx context.Context, id string) (*models.AdminUser, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, ErrNotFound
}

func (m *MockAdminUserRepository) GetAllStaff(ctx context.Context, roleFilter string, search string) ([]models.AdminUser, error) {
	if m.GetAllStaffFunc != nil {
		return m.GetAllStaffFunc(ctx, roleFilter, search)
	}
	return []models.AdminUser{}, nil
}

func (m *MockAdminUserRepository) CreateStaff(ctx context.Context, req *models.CreateStaffRequest, passwordHash string) (*models.AdminUser, error) {
	if m.CreateStaffFunc != nil {
		return m.CreateStaffFunc(ctx, req, passwordHash)
	}
	return nil, nil
}

func (m *MockAdminUserRepository) UpdateStaff(ctx context.Context, id string, req *models.UpdateStaffRequest) (*models.AdminUser, error) {
	if m.UpdateStaffFunc != nil {
		return m.UpdateStaffFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *MockAdminUserRepository) ToggleStaffActive(ctx context.Context, id string) (*models.AdminUser, error) {
	if m.ToggleStaffActiveFunc != nil {
		return m.ToggleStaffActiveFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockAdminUserRepository) ResetStaffPassword(ctx context.Context, id string, passwordHash string) error {
	if m.ResetStaffPasswordFunc != nil {
		return m.ResetStaffPasswordFunc(ctx, id, passwordHash)
	}
	return nil
}

func (m *MockAdminUserRepository) DeleteStaff(ctx context.Context, id string) error {
	if m.DeleteStaffFunc != nil {
		return m.DeleteStaffFunc(ctx, id)
	}
	return nil
}

func (m *MockAdminUserRepository) CountActiveSuperAdmins(ctx context.Context) (int, error) {
	if m.CountActiveSuperAdminsFunc != nil {
		return m.CountActiveSuperAdminsFunc(ctx)
	}
	return 1, nil
}

func (m *MockAdminUserRepository) CreateAuditLog(ctx context.Context, logEntry *models.AuditLog) error {
	if m.CreateAuditLogFunc != nil {
		return m.CreateAuditLogFunc(ctx, logEntry)
	}
	return nil
}

func (m *MockAdminUserRepository) GetAuditLogs(ctx context.Context, actionFilter, search string, limit int) ([]models.AuditLog, error) {
	if m.GetAuditLogsFunc != nil {
		return m.GetAuditLogsFunc(ctx, actionFilter, search, limit)
	}
	return []models.AuditLog{}, nil
}

type MockSecurityRepository struct {
	GetAuditLogsFilteredFunc func(ctx context.Context, filter models.AuditLogFilter) (*models.AuditLogPage, error)
	IsIPBlockedFunc          func(ctx context.Context, ip string) (*models.BlockedIP, error)
	BlockIPFunc              func(ctx context.Context, entry *models.BlockedIP) (*models.BlockedIP, error)
	UnblockIPFunc            func(ctx context.Context, idOrIP string) error
	ListBlockedIPsFunc       func(ctx context.Context, activeOnly bool) ([]models.BlockedIP, error)
}

func (m *MockSecurityRepository) GetAuditLogsFiltered(ctx context.Context, filter models.AuditLogFilter) (*models.AuditLogPage, error) {
	if m.GetAuditLogsFilteredFunc != nil {
		return m.GetAuditLogsFilteredFunc(ctx, filter)
	}
	return &models.AuditLogPage{Items: []models.AuditLog{}, Total: 0, Page: 1, Limit: 50}, nil
}

func (m *MockSecurityRepository) IsIPBlocked(ctx context.Context, ip string) (*models.BlockedIP, error) {
	if m.IsIPBlockedFunc != nil {
		return m.IsIPBlockedFunc(ctx, ip)
	}
	return nil, ErrNotFound
}

func (m *MockSecurityRepository) BlockIP(ctx context.Context, entry *models.BlockedIP) (*models.BlockedIP, error) {
	if m.BlockIPFunc != nil {
		return m.BlockIPFunc(ctx, entry)
	}
	return entry, nil
}

func (m *MockSecurityRepository) UnblockIP(ctx context.Context, idOrIP string) error {
	if m.UnblockIPFunc != nil {
		return m.UnblockIPFunc(ctx, idOrIP)
	}
	return nil
}

func (m *MockSecurityRepository) ListBlockedIPs(ctx context.Context, activeOnly bool) ([]models.BlockedIP, error) {
	if m.ListBlockedIPsFunc != nil {
		return m.ListBlockedIPsFunc(ctx, activeOnly)
	}
	return []models.BlockedIP{}, nil
}

type MockBroadcastRepository struct {
	CreateBroadcastFunc           func(ctx context.Context, record *models.EmailBroadcastRecord) error
	UpdateBroadcastProgressFunc   func(ctx context.Context, id string, sentCount, failedCount int, status string) error
	GetAllBroadcastsFunc          func(ctx context.Context) ([]models.EmailBroadcastRecord, error)
	GetBroadcastByIDFunc          func(ctx context.Context, id string) (*models.EmailBroadcastRecord, error)
	ResolveRecipientsFunc         func(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error)
	ResolveRecipientsFilteredFunc func(ctx context.Context, scope, eventID, categoryID string, excludePromoOptOut bool) ([]models.BroadcastRecipient, error)
	CreateBroadcastRecipientsFunc func(ctx context.Context, broadcastID string, recipients []models.BroadcastRecipient) error
	UpdateRecipientStatusFunc     func(ctx context.Context, broadcastID, email, status, errorMsg string) error
	GetBroadcastRecipientsFunc    func(ctx context.Context, broadcastID string) ([]models.BroadcastRecipientRecord, error)
	GetFailedRecipientsFunc       func(ctx context.Context, broadcastID string) ([]models.BroadcastRecipient, error)
	UnsubscribeByTokenFunc        func(ctx context.Context, token string) error
}

func (m *MockBroadcastRepository) CreateBroadcast(ctx context.Context, record *models.EmailBroadcastRecord) error {
	if m.CreateBroadcastFunc != nil {
		return m.CreateBroadcastFunc(ctx, record)
	}
	return nil
}

func (m *MockBroadcastRepository) UpdateBroadcastProgress(ctx context.Context, id string, sentCount, failedCount int, status string) error {
	if m.UpdateBroadcastProgressFunc != nil {
		return m.UpdateBroadcastProgressFunc(ctx, id, sentCount, failedCount, status)
	}
	return nil
}

func (m *MockBroadcastRepository) GetAllBroadcasts(ctx context.Context) ([]models.EmailBroadcastRecord, error) {
	if m.GetAllBroadcastsFunc != nil {
		return m.GetAllBroadcastsFunc(ctx)
	}
	return []models.EmailBroadcastRecord{}, nil
}

func (m *MockBroadcastRepository) GetBroadcastByID(ctx context.Context, id string) (*models.EmailBroadcastRecord, error) {
	if m.GetBroadcastByIDFunc != nil {
		return m.GetBroadcastByIDFunc(ctx, id)
	}
	return nil, ErrNotFound
}

func (m *MockBroadcastRepository) ResolveRecipients(ctx context.Context, scope, eventID string) ([]models.BroadcastRecipient, error) {
	if m.ResolveRecipientsFunc != nil {
		return m.ResolveRecipientsFunc(ctx, scope, eventID)
	}
	return []models.BroadcastRecipient{}, nil
}

func (m *MockBroadcastRepository) ResolveRecipientsFiltered(ctx context.Context, scope, eventID, categoryID string, excludePromoOptOut bool) ([]models.BroadcastRecipient, error) {
	if m.ResolveRecipientsFilteredFunc != nil {
		return m.ResolveRecipientsFilteredFunc(ctx, scope, eventID, categoryID, excludePromoOptOut)
	}
	return m.ResolveRecipients(ctx, scope, eventID)
}

func (m *MockBroadcastRepository) CreateBroadcastRecipients(ctx context.Context, broadcastID string, recipients []models.BroadcastRecipient) error {
	if m.CreateBroadcastRecipientsFunc != nil {
		return m.CreateBroadcastRecipientsFunc(ctx, broadcastID, recipients)
	}
	return nil
}

func (m *MockBroadcastRepository) UpdateRecipientStatus(ctx context.Context, broadcastID, email, status, errorMsg string) error {
	if m.UpdateRecipientStatusFunc != nil {
		return m.UpdateRecipientStatusFunc(ctx, broadcastID, email, status, errorMsg)
	}
	return nil
}

func (m *MockBroadcastRepository) GetBroadcastRecipients(ctx context.Context, broadcastID string) ([]models.BroadcastRecipientRecord, error) {
	if m.GetBroadcastRecipientsFunc != nil {
		return m.GetBroadcastRecipientsFunc(ctx, broadcastID)
	}
	return []models.BroadcastRecipientRecord{}, nil
}

func (m *MockBroadcastRepository) GetFailedRecipients(ctx context.Context, broadcastID string) ([]models.BroadcastRecipient, error) {
	if m.GetFailedRecipientsFunc != nil {
		return m.GetFailedRecipientsFunc(ctx, broadcastID)
	}
	return []models.BroadcastRecipient{}, nil
}

func (m *MockBroadcastRepository) UnsubscribeByToken(ctx context.Context, token string) error {
	if m.UnsubscribeByTokenFunc != nil {
		return m.UnsubscribeByTokenFunc(ctx, token)
	}
	return nil
}
