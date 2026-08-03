package models

import "time"

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

type CreateOrderRequest struct {
	EventID          string `json:"eventId"`
	TicketCategoryID string `json:"ticketCategoryId"`
	Quantity         int    `json:"quantity"`
	UserName         string `json:"userName"`
	UserEmail        string `json:"userEmail"`
}

type OrderRecord struct {
	ID            string    `json:"id"`
	OrderCode     string    `json:"orderCode"`
	UserID        string    `json:"userId,omitempty"`
	EventID       string    `json:"eventId"`
	EventTitle    string    `json:"eventTitle"`
	Artist        string    `json:"artist"`
	Venue         string    `json:"venue"`
	Date          string    `json:"date"`
	CategoryName  string    `json:"categoryName"`
	Quantity      int       `json:"quantity"`
	TotalPrice    float64   `json:"totalPrice"`
	UserName      string    `json:"userName"`
	UserEmail     string    `json:"userEmail"`
	QRCode        string    `json:"qrCode"`
	Status        string    `json:"status"`
	PaymentMethod string    `json:"paymentMethod"`
	CreatedAt     time.Time `json:"createdAt"`
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
	Code string `json:"code"`
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


