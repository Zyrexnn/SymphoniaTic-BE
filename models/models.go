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

