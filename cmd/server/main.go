package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/database"
	_ "github.com/Zyrexnn/SymphoniaTic-be/docs"
	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"
)

// @title SymphoniaTic REST API Documentation
// @version 1.0
// @description Interactive OpenAPI/Swagger UI documentation for SymphoniaTic Concert Ticketing Platform.
// @termsOfService http://swagger.io/terms/

// @contact.name SymphoniaTic API Support
// @contact.email support@symphoniatic.id

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8082
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer <your_token>" to authenticate user/admin endpoints.

func setupRouter(
	cfg *config.Config,
	eventRepo repositories.EventRepository,
	orderRepo repositories.OrderRepository,
	refundRepo repositories.RefundRepository,
) *fiber.App {
	eventCtrl := controllers.NewEventController(eventRepo)
	categoryCtrl := controllers.NewCategoryController(eventRepo)
	orderCtrl := controllers.NewOrderController(orderRepo)
	ticketCtrl := controllers.NewTicketController(orderRepo)
	refundCtrl := controllers.NewRefundController(refundRepo)
	adminCtrl := controllers.NewAdminController(orderRepo, &cfg.Auth)

	app := fiber.New(fiber.Config{
		AppName: "SymphoniaTic Native Golang REST API Server",
	})

	app.Use(logger.New())

	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:") ||
				strings.HasPrefix(origin, "https://localhost:") ||
				strings.HasPrefix(origin, "https://127.0.0.1:")
		},
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
	}))

	app.Static("/uploads", filepath.Clean("./uploads"))

	// Swagger UI Route
	app.Get("/swagger/*", swagger.HandlerDefault)

	api := app.Group("/api/v1")

	// Public Endpoints
	api.Get("/events", eventCtrl.GetEvents)
	api.Get("/events/:id", eventCtrl.GetEventByID)
	api.Get("/events/:id/seats", eventCtrl.GetEventSeats)
	api.Post("/orders", orderCtrl.CreateOrder)
	api.Get("/tickets/lookup", ticketCtrl.LookupTicketByCode)

	// Refund Routes
	api.Post("/refunds/request-otp", refundCtrl.RequestRefundOTP)
	api.Post("/refunds/submit", refundCtrl.SubmitRefund)
	api.Post("/refunds/status", refundCtrl.GetRefundStatus)

	// User Authentication Endpoints
	auth := api.Group("/auth")
	auth.Post("/register/request-otp", controllers.RequestRegisterOTP)
	auth.Post("/register/verify-otp", controllers.VerifyRegisterOTP)
	auth.Post("/login/password", controllers.PasswordLogin)
	auth.Post("/login/request-otp", controllers.RequestLoginOTP)
	auth.Post("/login/verify-otp", controllers.VerifyLoginOTP)
	auth.Post("/forgot-password/request-otp", controllers.RequestForgotPasswordOTP)
	auth.Post("/forgot-password/verify-otp", controllers.VerifyForgotPasswordOTP)
	auth.Post("/forgot-password/reset", controllers.ResetPassword)
	auth.Get("/me", middleware.RequireUserAuth, controllers.GetMyProfile)

	// Protected User Account Endpoints
	userGroup := api.Group("/user", middleware.RequireUserAuth)
	userGroup.Get("/orders", controllers.GetUserOrders)
	userGroup.Get("/orders/:orderCode", controllers.GetUserOrderByCode)
	userGroup.Put("/profile", controllers.UpdateUserProfile)
	userGroup.Post("/change-password", controllers.ChangeUserPassword)
	userGroup.Get("/dashboard-summary", controllers.GetUserDashboardSummary)
	userGroup.Get("/refunds", controllers.GetUserRefunds)

	admin := api.Group("/admin")
	admin.Post("/login", adminCtrl.AdminLogin)

	// Pemisahan middleware per-rute (alih-alih Group("")) agar otorisasi ADMIN tidak bocor memblokir rute PETUGAS di prefix /admin yang sama
	requireAdminOnly := middleware.RequireAdminAuth("ADMIN")
	requireGateOrAdmin := middleware.RequireAdminAuth("ADMIN", "PETUGAS")

	admin.Get("/dashboard", requireAdminOnly, adminCtrl.GetAdminDashboardMetrics)
	admin.Post("/upload", requireAdminOnly, adminCtrl.UploadImage)

	admin.Post("/events", requireAdminOnly, eventCtrl.CreateEvent)
	admin.Put("/events/:id", requireAdminOnly, eventCtrl.UpdateEvent)
	admin.Patch("/events/:id/toggle-close", requireAdminOnly, eventCtrl.ToggleEventClose)
	admin.Delete("/events/:id", requireAdminOnly, eventCtrl.DeleteEvent)

	admin.Post("/events/:id/categories", requireAdminOnly, categoryCtrl.CreateTicketCategory)
	admin.Put("/categories/:id", requireAdminOnly, categoryCtrl.UpdateTicketCategory)
	admin.Delete("/categories/:id", requireAdminOnly, categoryCtrl.DeleteTicketCategory)

	admin.Get("/orders", requireGateOrAdmin, orderCtrl.GetAllOrders)
	admin.Patch("/orders/:id/status", requireAdminOnly, orderCtrl.UpdateOrderStatus)
	admin.Post("/tickets/checkin", requireGateOrAdmin, ticketCtrl.CheckInTicket)
	admin.Get("/tickets/attendance-summary", requireGateOrAdmin, ticketCtrl.GetAttendanceSummary)

	admin.Get("/refunds", requireAdminOnly, refundCtrl.AdminGetAllRefunds)
	admin.Patch("/refunds/:id/status", requireAdminOnly, refundCtrl.AdminUpdateRefundStatus)

	return app
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Gagal memuat konfigurasi server: %v", err)
	}

	database.ConnectDBWithConfig(&cfg.Database)

	services.InitMailerWorkerPool(&cfg.SMTP, 5, 200)

	eventRepo := repositories.NewEventRepository(database.DB)
	orderRepo := repositories.NewOrderRepository(database.DB)
	refundRepo := repositories.NewRefundRepository(database.DB)

	controllers.InitDefaultControllers(database.DB, &cfg.Auth)

	app := setupRouter(cfg, eventRepo, orderRepo, refundRepo)

	serverShutdown := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Menerima sinyal terminasi, menghentikan server secara graceful...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Gagal menghentikan server Fiber: %v", err)
		}
		services.ShutdownMailer()
		if database.DB != nil {
			_ = database.DB.Close()
		}
		close(serverShutdown)
	}()

	log.Printf("SymphoniaTic REST API Server berjalan di http://localhost:%s", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server Fiber terhenti dengan error: %v", err)
	}
	<-serverShutdown
	log.Println("Server berhasil dihentikan dengan aman.")
}
