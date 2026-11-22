package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type TicketCategory struct {
	ID             string    `json:"id"`
	EventID        string    `json:"eventId"`
	Name           string    `json:"name"`
	Price          float64   `json:"price"`
	Quota          int       `json:"quota"`
	RemainingQuota int       `json:"remainingQuota"`
	CreatedAt      time.Time `json:"createdAt"`
}

type RundownItem struct {
	Time     string `json:"time"`
	Activity string `json:"activity"`
}

type EventItem struct {
	ID                 string           `json:"id"`
	Title              string           `json:"title"`
	Artist             string           `json:"artist"`
	Venue              string           `json:"venue"`
	Date               string           `json:"date"`
	Time               string           `json:"time"`
	Category           string           `json:"category"`
	CategoryBadgeColor string           `json:"categoryBadgeColor"`
	Image              string           `json:"image"`
	AudioURL           string           `json:"audioUrl"`
	Conductor          string           `json:"conductor"`
	OpenGate           string           `json:"openGate"`
	Address            string           `json:"address"`
	Organizer          string           `json:"organizer"`
	Subtitle           string           `json:"subtitle"`
	Description        string           `json:"description"`
	IsClosed           bool             `json:"isClosed"`
	EventDateTime      *time.Time       `json:"eventDateTime,omitempty"`
	Rundown            []RundownItem    `json:"rundown"`
	Categories         []TicketCategory `json:"categories"`
}

type SeatItem struct {
	ID         string `json:"id"`
	EventID    string `json:"eventId"`
	CategoryID string `json:"categoryId"`
	SeatCode   string `json:"seatCode"`
	RowLabel   string `json:"rowLabel"`
	SeatNumber int    `json:"seatNumber"`
	OrderID    string `json:"orderId,omitempty"`
	Status     string `json:"status"`
}

type TicketItem struct {
	SeatCode     string `json:"seatCode"`
	AttendeeName string `json:"attendeeName"`
	QRCode       string `json:"qrCode"`
	Status       string `json:"status"`
}

type CreateOrderRequest struct {
	EventID          string   `json:"eventId"`
	TicketCategoryID string   `json:"ticketCategoryId"`
	Quantity         int      `json:"quantity"`
	UserName         string   `json:"userName"`
	UserEmail        string   `json:"userEmail"`
	SeatNumbers      []string `json:"seatNumbers,omitempty"`
	AttendeeNames    []string `json:"attendeeNames,omitempty"`
}

type OrderRecord struct {
	ID             string       `json:"id"`
	OrderCode      string       `json:"orderCode"`
	UserID         string       `json:"userId,omitempty"`
	EventID        string       `json:"eventId"`
	EventTitle     string       `json:"eventTitle"`
	Artist         string       `json:"artist"`
	Venue          string       `json:"venue"`
	Date           string       `json:"date"`
	OpenGate       string       `json:"openGate,omitempty"`
	EventTime      string       `json:"eventTime,omitempty"`
	VenueAddress   string       `json:"venueAddress,omitempty"`
	CategoryName   string       `json:"categoryName"`
	Quantity       int          `json:"quantity"`
	TotalPrice     float64      `json:"totalPrice"`
	UserName       string       `json:"userName"`
	UserEmail      string       `json:"userEmail"`
	QRCode            string       `json:"qrCode"`
	SeatNumbers       string       `json:"seatNumbers"`
	AttendeeNames     string       `json:"attendeeNames,omitempty"`
	CheckedInSeats    string       `json:"checkedInSeats,omitempty"`
	CheckedInAt       *time.Time   `json:"checkedInAt,omitempty"`
	CheckedInSeat     string       `json:"checkedInSeat,omitempty"`
	CheckedInAttendee string       `json:"checkedInAttendee,omitempty"`
	Tickets           []TicketItem `json:"tickets,omitempty"`
	Status         string       `json:"status"`
	PaymentMethod  string       `json:"paymentMethod"`
	CreatedAt      time.Time    `json:"createdAt"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateCategoryInput struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Quota int     `json:"quota"`
}

type CreateEventRequest struct {
	Title              string                `json:"title"`
	Artist             string                `json:"artist"`
	Venue              string                `json:"venue"`
	Date               string                `json:"date"`
	Time               string                `json:"time"`
	Category           string                `json:"category"`
	CategoryBadgeColor string                `json:"categoryBadgeColor"`
	Image              string                `json:"image"`
	AudioURL           string                `json:"audioUrl"`
	Conductor          string                `json:"conductor"`
	OpenGate           string                `json:"openGate"`
	Address            string                `json:"address"`
	Organizer          string                `json:"organizer"`
	Subtitle           string                `json:"subtitle"`
	Description        string                `json:"description"`
	Rundown            []RundownItem         `json:"rundown"`
	Categories         []CreateCategoryInput `json:"categories"`
}

type UpdateEventRequest struct {
	Title              string        `json:"title"`
	Artist             string        `json:"artist"`
	Venue              string        `json:"venue"`
	Date               string        `json:"date"`
	Time               string        `json:"time"`
	Category           string        `json:"category"`
	CategoryBadgeColor string        `json:"categoryBadgeColor"`
	Image              string        `json:"image"`
	AudioURL           string        `json:"audioUrl"`
	Conductor          string        `json:"conductor"`
	OpenGate           string        `json:"openGate"`
	Address            string        `json:"address"`
	Organizer          string        `json:"organizer"`
	Subtitle           string        `json:"subtitle"`
	Description        string        `json:"description"`
	Rundown            []RundownItem `json:"rundown"`
}

type UpdateTicketCategoryRequest struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Quota int     `json:"quota"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

type CheckInRequest struct {
	Code           string `json:"code"`
	OrderCode      string `json:"orderCode,omitempty"`
	BypassOpenGate bool   `json:"bypassOpenGate,omitempty"`
}

// ─── REFUND MODELS ───

type RequestRefundOTPInput struct {
	OrderCode string `json:"orderCode"`
	UserEmail string `json:"userEmail"`
}

type SubmitRefundInput struct {
	OrderCode     string `json:"orderCode"`
	UserEmail     string `json:"userEmail"`
	OTPCode       string `json:"otpCode"`
	BankName      string `json:"bankName"`
	AccountNumber string `json:"accountNumber"`
	AccountHolder string `json:"accountHolder"`
	Reason        string `json:"reason"`
}

type RefundRequestRecord struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"orderId"`
	OrderCode     string    `json:"orderCode"`
	UserEmail     string    `json:"userEmail"`
	BankName      string    `json:"bankName"`
	AccountNumber string    `json:"accountNumber"`
	AccountHolder string    `json:"accountHolder"`
	Reason        string    `json:"reason"`
	RefundAmount  float64   `json:"refundAmount"`
	Status        string    `json:"status"`
	AdminNote     string    `json:"adminNote"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	// Joined details for UI/Admin
	EventTitle   string `json:"eventTitle,omitempty"`
	CategoryName string `json:"categoryName,omitempty"`
	Quantity     int    `json:"quantity,omitempty"`
	UserName     string `json:"userName,omitempty"`
}

type UpdateRefundStatusRequest struct {
	Status    string `json:"status"`
	AdminNote string `json:"adminNote"`
}

type CheckRefundStatusRequest struct {
	OrderCode string `json:"orderCode"`
	UserEmail string `json:"userEmail"`
}

// ─── USER & AUTHENTICATION MODELS ───

type UserRecord struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	Role      string    `json:"role"`
	IsVerified bool     `json:"isVerified"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AuthOTPRecord struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	OTPCode   string    `json:"otpCode"`
	Purpose   string    `json:"purpose"`
	Attempts  int       `json:"attempts"`
	IsUsed    bool      `json:"isUsed"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type RegisterRequestOTPInput struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type RegisterVerifyOTPInput struct {
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"`
	OTPCode  string `json:"otpCode"`
	Password string `json:"password"`
}

type LoginRequestOTPInput struct {
	Email string `json:"email"`
}

type LoginVerifyOTPInput struct {
	Email   string `json:"email"`
	OTPCode string `json:"otpCode"`
}

type PasswordLoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ForgotPasswordRequestOTPInput struct {
	Email string `json:"email"`
}

type ForgotPasswordVerifyOTPInput struct {
	Email   string `json:"email"`
	OTPCode string `json:"otpCode"`
}

type ResetPasswordInput struct {
	ResetToken  string `json:"resetToken"`
	NewPassword string `json:"newPassword"`
}

type AuthResponseData struct {
	Token string     `json:"token"`
	User  UserRecord `json:"user"`
}

type VerifyOTPResponseData struct {
	ResetToken string `json:"resetToken,omitempty"`
	Message    string `json:"message"`
}

type UpdateProfileInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type UserDashboardSummary struct {
	TotalTicketsBought  int `json:"totalTicketsBought"`
	UpcomingEventsCount int `json:"upcomingEventsCount"`
	PastEventsCount     int `json:"pastEventsCount"`
	ActiveRefundsCount  int `json:"activeRefundsCount"`
}
type EventRevenueStat struct {
	EventID     string  `json:"eventId"`
	Title       string  `json:"title"`
	Revenue     float64 `json:"revenue"`
	TicketsSold int     `json:"ticketsSold"`
}

type TimelineStat struct {
	Month   string  `json:"month"`
	Revenue float64 `json:"revenue"`
	Tickets int     `json:"tickets"`
}

type CategoryStat struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type RecentOrderSummary struct {
	ID         string    `json:"id"`
	OrderCode  string    `json:"orderCode"`
	EventTitle string    `json:"eventTitle"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"totalPrice"`
	UserName   string    `json:"userName"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

type AdminDashboardMetrics struct {
	TotalRevenue         float64              `json:"totalRevenue"`
	TicketsSold          int                  `json:"ticketsSold"`
	RemainingQuota       int                  `json:"remainingQuota"`
	TotalEvents          int                  `json:"totalEvents"`
	TotalOrders          int                  `json:"totalOrders"`
	EventStats           []EventRevenueStat   `json:"eventStats"`
	RecentOrders         []RecentOrderSummary `json:"recentOrders"`
	RevenueTimeline      []TimelineStat       `json:"revenueTimeline"`
	CategoryDistribution []CategoryStat       `json:"categoryDistribution"`
}

type GateAttendanceSummary struct {
	TotalTicketsIssued int     `json:"totalTicketsIssued"`
	TotalCheckedIn     int     `json:"totalCheckedIn"`
	RemainingCheckIn   int     `json:"remainingCheckIn"`
	CheckInPercentage  float64 `json:"checkInPercentage"`
}

func ParseCommaList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func ResolveAttendeeNames(provided []string, fallbackName string, count int) []string {
	if count <= 0 {
		return nil
	}
	fallback := strings.TrimSpace(fallbackName)
	resolved := make([]string, count)
	for i := 0; i < count; i++ {
		if i < len(provided) && strings.TrimSpace(provided[i]) != "" {
			resolved[i] = strings.TrimSpace(provided[i])
		} else {
			resolved[i] = fallback
		}
	}
	return resolved
}

func EncodeAttendeeNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	b, err := json.Marshal(names)
	if err != nil {
		return ""
	}
	return string(b)
}

func DecodeAttendeeNames(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var parsed []string
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			return parsed
		}
	}
	return ParseCommaList(trimmed)
}

// BuildOrderTickets menyusun daftar pass per kursi beserta status check-in dan nama pengunjung masing-masing
func BuildOrderTickets(record *OrderRecord) {
	if record == nil {
		return
	}
	seats := ParseCommaList(record.SeatNumbers)
	count := len(seats)
	if count == 0 {
		count = record.Quantity
	}
	if count <= 0 {
		return
	}

	attendees := ResolveAttendeeNames(DecodeAttendeeNames(record.AttendeeNames), record.UserName, count)
	checkedSet := make(map[string]bool)
	for _, s := range ParseCommaList(record.CheckedInSeats) {
		checkedSet[strings.ToUpper(s)] = true
	}

	isAllCheckedIn := strings.ToUpper(strings.TrimSpace(record.Status)) == "CHECKED_IN"
	baseQR := strings.TrimSpace(record.QRCode)
	if baseQR == "" && record.OrderCode != "" {
		baseQR = "QR-" + record.OrderCode
	}

	tickets := make([]TicketItem, 0, count)
	for i := 0; i < count; i++ {
		seatCode := ""
		if i < len(seats) {
			seatCode = seats[i]
		}
		qr := baseQR
		if seatCode != "" && record.OrderCode != "" {
			qr = fmt.Sprintf("QR-%s-%s", record.OrderCode, seatCode)
		}

		itemStatus := record.Status
		if isAllCheckedIn || (seatCode != "" && checkedSet[strings.ToUpper(seatCode)]) {
			itemStatus = "CHECKED_IN"
		}

		tickets = append(tickets, TicketItem{
			SeatCode:     seatCode,
			AttendeeName: attendees[i],
			QRCode:       qr,
			Status:       itemStatus,
		})
	}
	record.Tickets = tickets
}

// ParseSeatCheckins membaca peta JSON stempel waktu check-in per kursi.
func ParseSeatCheckins(raw string) map[string]string {
	result := make(map[string]string)
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return result
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
		for k, v := range parsed {
			key := strings.ToUpper(strings.TrimSpace(k))
			if key != "" && strings.TrimSpace(v) != "" {
				result[key] = strings.TrimSpace(v)
			}
		}
	}
	return result
}

// EncodeSeatCheckins menyerialisasi peta stempel waktu check-in per kursi ke JSON.
func EncodeSeatCheckins(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ResolveTicketAttendee mengambil nama pemegang kursi tertentu dari metadata tiket pesanan.
func ResolveTicketAttendee(record *OrderRecord, targetSeat string) string {
	if record == nil {
		return ""
	}
	if len(record.Tickets) == 0 {
		BuildOrderTickets(record)
	}
	cleanSeat := strings.ToUpper(strings.TrimSpace(targetSeat))
	if cleanSeat != "" {
		for _, t := range record.Tickets {
			if strings.EqualFold(strings.TrimSpace(t.SeatCode), cleanSeat) && strings.TrimSpace(t.AttendeeName) != "" {
				return strings.TrimSpace(t.AttendeeName)
			}
		}
	}
	return strings.TrimSpace(record.UserName)
}
