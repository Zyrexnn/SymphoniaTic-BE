package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/database"
	_ "github.com/Zyrexnn/SymphoniaTic-be/docs"
	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
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

func newRateLimiter(maxRequests int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: expiration,
		LimitReached: func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusTooManyRequests).JSON(models.APIResponse{
				Success: false,
				Message: "Terlalu banyak permintaan. Silakan tunggu beberapa saat lagi.",
			})
		},
	})
}

func setupRouter(
	cfg *config.Config,
	eventRepo repositories.EventRepository,
	orderRepo repositories.OrderRepository,
	refundRepo repositories.RefundRepository,
	promoRepo ...repositories.PromoRepository,
) *fiber.App {
	var activePromoRepo repositories.PromoRepository = &repositories.MockPromoRepository{}
	if len(promoRepo) > 0 && promoRepo[0] != nil {
		activePromoRepo = promoRepo[0]
	}
	return setupRouterWithStaff(cfg, eventRepo, orderRepo, refundRepo, activePromoRepo, &repositories.MockAdminUserRepository{})
}

func setupRouterWithStaff(
	cfg *config.Config,
	eventRepo repositories.EventRepository,
	orderRepo repositories.OrderRepository,
	refundRepo repositories.RefundRepository,
	promoRepo repositories.PromoRepository,
	staffRepo repositories.AdminUserRepository,
	secRepo ...repositories.SecurityRepository,
) *fiber.App {
	eventCtrl := controllers.NewEventController(eventRepo)
	categoryCtrl := controllers.NewCategoryController(eventRepo)
	orderCtrl := controllers.NewOrderController(orderRepo)
	ticketCtrl := controllers.NewTicketController(orderRepo)
	refundCtrl := controllers.NewRefundControllerWithOrderRepo(refundRepo, orderRepo)
	adminCtrl := controllers.NewAdminControllerWithStaffRepo(orderRepo, staffRepo, &cfg.Auth)
	staffCtrl := controllers.NewStaffController(staffRepo)
	activePromoRepo := promoRepo
	if activePromoRepo == nil {
		activePromoRepo = &repositories.MockPromoRepository{}
	}
	var activeBroadcastRepo repositories.BroadcastRepository = &repositories.MockBroadcastRepository{}
	if database.DB != nil {
		activeBroadcastRepo = repositories.NewBroadcastRepository(database.DB)
	}
	eventCtrl.SetBroadcastRepo(activeBroadcastRepo)
	broadcastCtrl := controllers.NewBroadcastController(activeBroadcastRepo, eventRepo)
	promoCtrl := controllers.NewPromoController(activePromoRepo)

	var activeSecRepo repositories.SecurityRepository = &repositories.MockSecurityRepository{}
	if len(secRepo) > 0 && secRepo[0] != nil {
		activeSecRepo = secRepo[0]
	}
	blockMgr := middleware.NewIPBlocklistManager(activeSecRepo, staffRepo)
	if len(secRepo) > 0 && secRepo[0] != nil {
		if err := blockMgr.RefreshFromDB(context.Background()); err != nil {
			log.Printf("Peringatan: gagal memuat daftar blokir IP awal: %v", err)
		}
	}
	securityCtrl := controllers.NewSecurityController(activeSecRepo, staffRepo, blockMgr)

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
	api.Use(blockMgr.BlocklistMiddleware())
	api.Use(blockMgr.SecurityAuditMiddleware())

	authRateLimiter := newRateLimiter(10, 1*time.Minute)
	lookupRateLimiter := newRateLimiter(30, 1*time.Minute)
	orderRateLimiter := newRateLimiter(15, 1*time.Minute)

	// Public Endpoints
	api.Get("/events", eventCtrl.GetEvents)
	api.Get("/events/:id", eventCtrl.GetEventByID)
	api.Get("/events/:id/seats", eventCtrl.GetEventSeats)
	api.Post("/orders", orderRateLimiter, orderCtrl.CreateOrder)
	api.Get("/promos/active", lookupRateLimiter, promoCtrl.GetActivePromos)
	api.Post("/promos/validate", lookupRateLimiter, promoCtrl.ValidatePromo)
	api.Get("/tickets/lookup", lookupRateLimiter, ticketCtrl.LookupTicketByCode)

	// Refund Routes
	api.Post("/refunds/request-otp", authRateLimiter, refundCtrl.RequestRefundOTP)
	api.Post("/refunds/submit", authRateLimiter, refundCtrl.SubmitRefund)
	api.Post("/refunds/status", lookupRateLimiter, refundCtrl.GetRefundStatus)

	// User Authentication Endpoints
	auth := api.Group("/auth")
	auth.Post("/register/request-otp", authRateLimiter, controllers.RequestRegisterOTP)
	auth.Post("/register/verify-otp", authRateLimiter, controllers.VerifyRegisterOTP)
	auth.Post("/login/password", authRateLimiter, controllers.PasswordLogin)
	auth.Post("/login/request-otp", authRateLimiter, controllers.RequestLoginOTP)
	auth.Post("/login/verify-otp", authRateLimiter, controllers.VerifyLoginOTP)
	auth.Post("/forgot-password/request-otp", authRateLimiter, controllers.RequestForgotPasswordOTP)
	auth.Post("/forgot-password/verify-otp", authRateLimiter, controllers.VerifyForgotPasswordOTP)
	auth.Post("/forgot-password/reset", authRateLimiter, controllers.ResetPassword)
	auth.Get("/me", middleware.RequireUserAuth, controllers.GetMyProfile)

	// Protected User Account Endpoints
	userGroup := api.Group("/user", middleware.RequireUserAuth)
	userGroup.Get("/orders", controllers.GetUserOrders)
	userGroup.Get("/orders/:orderCode", controllers.GetUserOrderByCode)
	userGroup.Post("/orders/:orderCode/refund", refundCtrl.SubmitUserRefund)
	userGroup.Put("/profile", controllers.UpdateUserProfile)
	userGroup.Post("/change-password", controllers.ChangeUserPassword)
	userGroup.Get("/dashboard-summary", controllers.GetUserDashboardSummary)
	userGroup.Get("/refunds", controllers.GetUserRefunds)

	admin := api.Group("/admin")
	admin.Use(controllers.AuditTrailMiddleware(staffRepo))
	admin.Post("/login", authRateLimiter, adminCtrl.AdminLogin)

	// Pemisahan middleware per-rute (alih-alih Group("")) agar otorisasi ADMIN tidak bocor memblokir rute PETUGAS di prefix /admin yang sama
	requireSuperAdminOnly := middleware.RequireAdminAuth("SUPER_ADMIN")
	requireAdminOnly := middleware.RequireAdminAuth("ADMIN")
	requireGateOrAdmin := middleware.RequireAdminAuth("ADMIN", "PETUGAS")

	admin.Get("/staff", requireSuperAdminOnly, staffCtrl.GetAllStaff)
	admin.Post("/staff", requireSuperAdminOnly, staffCtrl.CreateStaff)
	admin.Put("/staff/:id", requireSuperAdminOnly, staffCtrl.UpdateStaff)
	admin.Patch("/staff/:id/status", requireSuperAdminOnly, staffCtrl.ToggleStaffStatus)
	admin.Post("/staff/:id/reset-password", requireSuperAdminOnly, staffCtrl.ResetStaffPassword)
	admin.Delete("/staff/:id", requireSuperAdminOnly, staffCtrl.DeleteStaff)
	admin.Get("/audit-logs", requireSuperAdminOnly, staffCtrl.GetAuditLogs)
	admin.Get("/audit-logs/search", requireSuperAdminOnly, securityCtrl.GetFilteredAuditLogs)
	admin.Get("/security/blocked-ips", requireSuperAdminOnly, securityCtrl.ListBlockedIPs)
	admin.Post("/security/blocked-ips", requireSuperAdminOnly, securityCtrl.BlockIP)
	admin.Delete("/security/blocked-ips/:id", requireSuperAdminOnly, securityCtrl.UnblockIP)

	admin.Get("/dashboard", requireAdminOnly, adminCtrl.GetAdminDashboardMetrics)
	admin.Post("/upload", requireAdminOnly, adminCtrl.UploadImage)

	admin.Post("/events", requireAdminOnly, eventCtrl.CreateEvent)
	admin.Put("/events/:id", requireAdminOnly, eventCtrl.UpdateEvent)
	admin.Patch("/events/:id/toggle-close", requireAdminOnly, eventCtrl.ToggleEventClose)
	admin.Post("/events/:id/cancel", requireAdminOnly, eventCtrl.CancelEvent)
	admin.Post("/events/:id/reschedule", requireAdminOnly, eventCtrl.RescheduleEvent)
	admin.Post("/events/:id/batch-refund", requireAdminOnly, refundCtrl.AdminBatchRefundByEvent)
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

	admin.Get("/promos", requireAdminOnly, promoCtrl.AdminGetAllPromos)
	admin.Post("/promos", requireAdminOnly, promoCtrl.AdminCreatePromo)
	admin.Put("/promos/:id", requireAdminOnly, promoCtrl.AdminUpdatePromo)
	admin.Patch("/promos/:id/toggle-active", requireAdminOnly, promoCtrl.AdminTogglePromoActive)
	admin.Delete("/promos/:id", requireAdminOnly, promoCtrl.AdminDeletePromo)

	admin.Get("/broadcasts", requireAdminOnly, broadcastCtrl.GetAllBroadcasts)
	admin.Post("/broadcasts", requireAdminOnly, broadcastCtrl.CreateBroadcast)
	admin.Get("/broadcasts/:id/recipients", requireAdminOnly, broadcastCtrl.GetBroadcastRecipients)
	admin.Post("/broadcasts/:id/retry", requireAdminOnly, broadcastCtrl.RetryFailedBroadcast)

	api.Get("/unsubscribe", broadcastCtrl.UnsubscribePromo)
	api.Post("/unsubscribe", broadcastCtrl.UnsubscribePromo)

	return app
}

func validateServerConfig(cfg *config.Config) error {
	if cfg == nil || strings.TrimSpace(cfg.Auth.JWTSecret) == "" {
		return fmt.Errorf("variabel lingkungan JWT_SECRET wajib diisi sebelum menjalankan server")
	}
	return nil
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Gagal memuat konfigurasi server: %v", err)
	}
	if err := validateServerConfig(cfg); err != nil {
		log.Fatalf("Konfigurasi keamanan server tidak valid: %v", err)
	}

	database.ConnectDBWithConfig(&cfg.Database)

	services.InitMailerWorkerPool(&cfg.SMTP, 5, 200)
	schedulerCtx, stopScheduler := context.WithCancel(context.Background())
	services.StartEventCompletionScheduler(schedulerCtx, database.DB, 15*time.Minute)

	eventRepo := repositories.NewEventRepository(database.DB)
	orderRepo := repositories.NewOrderRepository(database.DB)
	refundRepo := repositories.NewRefundRepository(database.DB)
	promoRepo := repositories.NewPromoRepository(database.DB)
	staffRepo := repositories.NewAdminUserRepository(database.DB)
	secRepo := repositories.NewSecurityRepository(database.DB)

	controllers.InitDefaultControllers(database.DB, &cfg.Auth)

	app := setupRouterWithStaff(cfg, eventRepo, orderRepo, refundRepo, promoRepo, staffRepo, secRepo)

	serverShutdown := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Menerima sinyal terminasi, menghentikan server secara graceful...")
		stopScheduler()
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
