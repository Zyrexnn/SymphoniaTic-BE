package main

import (
	"log"
	"os"
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	database.ConnectDB()

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

	api := app.Group("/api/v1")

	// ─── Public Endpoints ───
	api.Get("/events", controllers.GetEvents)
	api.Get("/events/:id", controllers.GetEventByID)
	api.Post("/orders", controllers.CreateOrder)
	api.Get("/tickets/lookup", controllers.LookupTicketByCode)

	// ─── Admin Endpoints ───
	admin := api.Group("/admin")
	admin.Post("/login", controllers.AdminLogin)
	admin.Get("/dashboard", controllers.GetAdminDashboardMetrics)

	// Admin Event CRUD Routes
	admin.Post("/events", controllers.CreateEvent)
	admin.Put("/events/:id", controllers.UpdateEvent)
	admin.Delete("/events/:id", controllers.DeleteEvent)

	// Admin Category CRUD Routes
	admin.Post("/events/:id/categories", controllers.CreateTicketCategory)
	admin.Put("/categories/:id", controllers.UpdateTicketCategory)
	admin.Delete("/categories/:id", controllers.DeleteTicketCategory)

	// Admin Orders Management Routes
	admin.Get("/orders", controllers.GetAllOrders)
	admin.Patch("/orders/:id/status", controllers.UpdateOrderStatus)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("🚀 SymphoniaTic REST API Server berjalan di http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}
