package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

var (
	ErrNotFound          = errors.New("data tidak ditemukan")
	ErrEventClosed       = errors.New("penjualan tiket untuk pertunjukan ini telah ditutup karena konser sudah dimulai")
	ErrEventHasOrders    = errors.New("konser tidak dapat dihapus karena sudah memiliki transaksi tiket aktif")
	ErrQuotaExceeded     = errors.New("kuota tiket tidak mencukupi")
	ErrOTPCooldown       = errors.New("mohon tunggu 60 detik sebelum meminta OTP baru")
	ErrOTPInvalid        = errors.New("kode OTP tidak valid atau telah kedaluwarsa")
	ErrOTPMaxExceed      = errors.New("batas percobaan OTP telah habis, silakan ajukan OTP baru")
	ErrUnauthorized      = errors.New("tidak memiliki hak akses")
	ErrTicketAlreadyUsed = errors.New("tiket ini sudah pernah digunakan untuk check-in sebelumnya")
	ErrTicketInvalid     = errors.New("tiket tidak valid karena telah dibatalkan atau di-refund")
	ErrTicketNotIssued   = errors.New("status tiket tidak valid untuk check-in")
	ErrEventNotStarted        = errors.New("gerbang masuk (open gate) untuk konser ini belum dibuka")
	ErrEventExpired           = errors.New("tiket tidak dapat digunakan karena acara konser telah selesai")
	ErrSeatUnavailable        = errors.New("salah satu kursi yang dipilih sudah dipesan oleh penonton lain")
	ErrInvalidSeatSelection   = errors.New("pilihan nomor kursi tidak valid atau tidak sesuai dengan jumlah dan kategori tiket")
	ErrCategoryHasBookedSeats = errors.New("kategori tidak dapat dihapus karena sudah memiliki kursi yang terjual")
	ErrQuotaBelowBooked       = errors.New("kuota baru tidak boleh lebih kecil dari jumlah kursi yang sudah terjual")
	ErrPromoInvalid           = errors.New("kode promo tidak ditemukan atau tidak valid")
	ErrPromoExpired           = errors.New("kode promo belum berlaku atau sudah kedaluwarsa")
	ErrPromoQuotaExceeded     = errors.New("kuota penggunaan kode promo sudah habis")
	ErrPromoMinOrderNotMet    = errors.New("minimal transaksi belum terpenuhi untuk menggunakan kode promo ini")
	ErrPromoAlreadyUsed       = errors.New("anda sudah pernah menggunakan kode promo ini sebelumnya")
	ErrPromoMemberOnly        = errors.New("kode promo hanya dapat digunakan oleh pengguna yang telah login")
	ErrPromoDuplicateCode     = errors.New("kode promo sudah digunakan, silakan pilih kode lain")
	ErrPromoInactive          = errors.New("kode promo sedang dinonaktifkan")
	ErrRefundAlreadyExists    = errors.New("pengajuan refund untuk pesanan ini sudah ada")
)

type ErrOTPWrongCode struct {
	RemainingAttempts int
}

func (e *ErrOTPWrongCode) Error() string {
	return fmt.Sprintf("Kode OTP salah. Sisa percobaan: %d", e.RemainingAttempts)
}

type EventRepository interface {
	GetAllEvents(ctx context.Context) ([]models.EventItem, error)
	GetEventByID(ctx context.Context, id string) (*models.EventItem, error)
	CreateEvent(ctx context.Context, req *models.CreateEventRequest) (*models.EventItem, error)
	UpdateEvent(ctx context.Context, id string, req *models.UpdateEventRequest) error
	DeleteEvent(ctx context.Context, id string) error
	ToggleEventClose(ctx context.Context, id string) (bool, error)
	AddCategory(ctx context.Context, eventID string, req *models.CreateCategoryInput) (*models.TicketCategory, error)
	UpdateCategory(ctx context.Context, catID string, req *models.UpdateTicketCategoryRequest) error
	DeleteCategory(ctx context.Context, catID string) error
	GetCategoriesByEventIDs(ctx context.Context, eventIDs []string) (map[string][]models.TicketCategory, error)
	GetCategoriesByEventID(ctx context.Context, eventID string) ([]models.TicketCategory, error)
	GetSeatsByEventID(ctx context.Context, eventID string) ([]models.SeatItem, error)
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, req *models.CreateOrderRequest, authenticatedUserID string) (*models.OrderRecord, error)
	GetOrderByCode(ctx context.Context, code string) (*models.OrderRecord, error)
	GetOrderByID(ctx context.Context, id string) (*models.OrderRecord, error)
	LookupTicketByCode(ctx context.Context, code string) (*models.OrderRecord, error)
	CheckInTicket(ctx context.Context, orderCode string, bypassOpenGate bool) (*models.OrderRecord, error)
	GetUserOrders(ctx context.Context, userID, userEmail, statusFilter string) ([]models.OrderRecord, error)
	GetUserOrderByCode(ctx context.Context, code, userID, userEmail string) (*models.OrderRecord, error)
	GetAllOrders(ctx context.Context, searchQuery, statusFilter string) ([]models.OrderRecord, error)
	UpdateOrderStatus(ctx context.Context, id, status string) error
	GetAdminDashboardMetrics(ctx context.Context) (*models.AdminDashboardMetrics, error)
	GetAttendanceSummary(ctx context.Context, eventID string) (*models.GateAttendanceSummary, error)
	GetUserDashboardSummary(ctx context.Context, userID, userEmail string) (*models.UserDashboardSummary, error)
}

type RefundRepository interface {
	RequestRefundOTP(ctx context.Context, orderCode, userEmail, otpCode string, expiresAt time.Time, refundAmount float64, orderID string) error
	GetRefundByOrderCode(ctx context.Context, orderCode string) (*models.RefundRequestRecord, error)
	GetRefundWithOTP(ctx context.Context, orderCode string) (*models.RefundRequestRecord, string, time.Time, error)
	GetRefundStatus(ctx context.Context, orderCode, userEmail string) (*models.RefundRequestRecord, string, error)
	SubmitRefund(ctx context.Context, input *models.SubmitRefundInput) (*models.RefundRequestRecord, error)
	SubmitDirectUserRefund(ctx context.Context, order *models.OrderRecord, input *models.SubmitUserRefundInput) (*models.RefundRequestRecord, error)
	GetAllRefunds(ctx context.Context) ([]models.RefundRequestRecord, error)
	GetUserRefunds(ctx context.Context, userEmail string) ([]models.RefundRequestRecord, error)
	UpdateRefundStatus(ctx context.Context, refundID, newStatus, adminNote string) (*models.RefundRequestRecord, error)
}

type PromoRepository interface {
	GetPromoByCode(ctx context.Context, code string) (*models.Promotion, error)
	HasUserUsedPromo(ctx context.Context, promotionID, userID string) (bool, error)
	GetAllPromos(ctx context.Context) ([]models.Promotion, error)
	GetActivePromos(ctx context.Context) ([]models.Promotion, error)
	CreatePromo(ctx context.Context, req *models.CreatePromoRequest) (*models.Promotion, error)
	UpdatePromo(ctx context.Context, id string, req *models.UpdatePromoRequest) (*models.Promotion, error)
	TogglePromoActive(ctx context.Context, id string) (bool, error)
	DeletePromo(ctx context.Context, id string) error
}

type UserRepository interface {
	GetUserByID(ctx context.Context, id string) (*models.UserRecord, error)
	GetUserByEmail(ctx context.Context, email string) (*models.UserRecord, error)
	GetUserWithPassword(ctx context.Context, email string) (*models.UserRecord, string, error)
	GetPasswordHash(ctx context.Context, id string) (string, error)
	CreateUser(ctx context.Context, user *models.UserRecord, passwordHash string) error
	UpdateProfile(ctx context.Context, id string, name, phone string) (*models.UserRecord, error)
	UpdatePassword(ctx context.Context, id string, newPasswordHash string) error
	UpdatePasswordByEmail(ctx context.Context, email string, newPasswordHash string) error
	CheckCooldown(ctx context.Context, email, purpose string, cooldown time.Duration) (bool, error)
	IssueOTP(ctx context.Context, email, purpose, otpCode string, expiresAt time.Time, cooldown time.Duration) error
	VerifyOTP(ctx context.Context, email, purpose, otpCode string, maxAttempts int) error
}
