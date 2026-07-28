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
		AppName: "SymphoniaTic Native Golang REST API",
	})

	app.Use(logger.New())

	corsOrigins := os.Getenv("CORS_ORIGIN")
	if corsOrigins == "" {
		corsOrigins = "*"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     strings.Join([]string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH", "OPTIONS"}, ","),
		AllowCredentials: true,
	}))

	api := app.Group("/api/v1")

	// Public Routes
	api.Get("/events", controllers.GetEvents)
	api.Get("/events/:id", controllers.GetEventByID)
	api.Post("/orders", controllers.CreateOrder)
	api.Get("/tickets/lookup", controllers.LookupTicketByCode)

	// Admin Metrics Route
	api.Get("/admin/dashboard", controllers.GetAdminDashboardMetrics)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("🚀 SymphoniaTic REST API Backend berjalan di http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}
