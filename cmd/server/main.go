package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/database"
	_ "github.com/Zyrexnn/SymphoniaTic-be/docs"
	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
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

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Gagal memuat konfigurasi server: %v", err)
	}

	database.ConnectDBWithConfig(&cfg.Database)

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

	app.Static("/uploads", "./uploads")

	// Swagger UI Route
	app.Get("/swagger/*", swagger.HandlerDefault)

	api := app.Group("/api/v1")

	// Public Endpoints
	api.Get("/events", controllers.GetEvents)
	api.Get("/events/:id", controllers.GetEventByID)
	api.Post("/orders", controllers.CreateOrder)
	api.Get("/tickets/lookup", controllers.LookupTicketByCode)

	// Refund Routes
	api.Post("/refunds/request-otp", controllers.RequestRefundOTP)
	api.Post("/refunds/submit", controllers.SubmitRefund)
	api.Post("/refunds/status", controllers.GetRefundStatus)

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

	// Admin Endpoints
	admin := api.Group("/admin")
	admin.Post("/login", controllers.AdminLogin)

	adminProtected := admin.Group("", middleware.RequireAdminAuth("ADMIN", "PETUGAS"))
	adminProtected.Get("/dashboard", controllers.GetAdminDashboardMetrics)
	adminProtected.Post("/upload", controllers.UploadImage)

	// Admin Event CRUD Routes
	adminProtected.Post("/events", controllers.CreateEvent)
	adminProtected.Put("/events/:id", controllers.UpdateEvent)
	adminProtected.Patch("/events/:id/toggle-close", controllers.ToggleEventClose)
	adminProtected.Delete("/events/:id", controllers.DeleteEvent)

	// Admin Category CRUD Routes
	adminProtected.Post("/events/:id/categories", controllers.CreateTicketCategory)
	adminProtected.Put("/categories/:id", controllers.UpdateTicketCategory)
	adminProtected.Delete("/categories/:id", controllers.DeleteTicketCategory)

	// Admin Orders Management Routes
	adminProtected.Get("/orders", controllers.GetAllOrders)
	adminProtected.Patch("/orders/:id/status", controllers.UpdateOrderStatus)

	// Admin Refund Management Routes
	adminProtected.Get("/refunds", controllers.AdminGetAllRefunds)
	adminProtected.Patch("/refunds/:id/status", controllers.AdminUpdateRefundStatus)

	serverShutdown := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Menerima sinyal terminasi, menghentikan server secara graceful...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Gagal menghentikan server Fiber: %v", err)
		}
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
