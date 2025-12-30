package main

import (
	"dramabang/database"
	"dramabang/handlers"
	"dramabang/models"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	app := fiber.New()

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Serve Uploads via Backend
	// In Docker, files are in public/uploads (mounted volume)
	app.Static("/uploads", "public/uploads")

	// Database
	database.Connect()
	models.MigrateDramas(database.DB)
	models.MigrateSettings(database.DB)
	models.MigrateUsers(database.DB)
	models.MigrateHistory(database.DB)
	models.MigrateLogs(database.DB)
	// Migrate Comments & Bookmarks
	database.DB.AutoMigrate(&models.Comment{})
	models.MigrateBookmarks(database.DB)

	// Routes
	api := app.Group("/api")

	// User endpoints
	api.Get("/trending", handlers.GetTrending)
	api.Get("/latest", handlers.GetLatest)
	api.Get("/search", handlers.GetSearch)
	api.Get("/detail", handlers.GetDetail)
	api.Get("/stream", handlers.GetStream)
	api.Get("/random", handlers.GetRandom)
	api.Get("/hero", handlers.GetHero)
	api.Get("/settings", handlers.GetPublicSettings)
	api.Get("/sitemap", handlers.GetSitemapData)
	api.Post("/auth/google", handlers.VerifyGoogleToken)
	api.Post("/auth/login", handlers.LocalLogin)
	api.Put("/user/profile", handlers.UpdateUserProfile) // New Profile Update
	api.Put("/user/password", handlers.UpdatePassword)   // New Password Change

	// My List (Bookmarks)
	api.Get("/mylist", handlers.GetBookmarks)
	api.Post("/mylist", handlers.AddBookmark)
	api.Delete("/mylist/:bookId", handlers.RemoveBookmark)
	api.Get("/mylist/check/:bookId", handlers.CheckBookmark)

	// Comments
	api.Get("/comments/:bookId", handlers.GetComments)
	api.Post("/comments", handlers.PostComment)
	api.Put("/comments/:id", handlers.UpdateComment)
	api.Delete("/comments/:id", handlers.DeleteComment)

	// History
	api.Post("/history", handlers.SaveHistory)
	api.Get("/history", handlers.GetHistory)

	// Admin Login (Public)
	api.Post("/admin/login", handlers.AdminLogin)

	// Admin config (Protected)
	admin := api.Group("/admin")
	admin.Use(func(c *fiber.Ctx) error {
		// Get token from header
		token := c.Get("Authorization")
		if token != "admin-secret-token-123" {
			return c.Status(401).JSON(fiber.Map{"status": "error", "message": "Unauthorized"})
		}
		return c.Next()
	})

	admin.Get("/dramas", handlers.GetAdminDramas)
	admin.Put("/dramas/:id", handlers.UpdateDrama)
	admin.Put("/dramas/:id/feature", handlers.ToggleFeatured)
	admin.Delete("/dramas/:id", handlers.DeleteDrama)
	admin.Post("/action/ingest", handlers.TriggerIngest)
	admin.Post("/action/dedup", handlers.TriggerDedup)
	admin.Get("/logs", handlers.GetSystemLogs)

	// Settings
	admin.Get("/settings", handlers.GetSettings)
	app.Post("/api/admin/settings", handlers.UpdateSettings)
	app.Post("/api/admin/upload", handlers.UploadFile) // New Upload Route

	// User Admin
	// User Admin (Protected)
	admin.Get("/users", handlers.GetAdminUsers)
	admin.Delete("/users/:id", handlers.DeleteUser)
	admin.Get("/users/:id/stats", handlers.GetUserStats)
	admin.Put("/users/:id/role", handlers.UpdateUserRole)

	// Melolo API Routes (Integrated)
	// Routes are now handled by the Universal Adapter via standard endpoints (/api/detail, etc.)
	// Legacy routes removed.

	log.Println("Starting server on :3000...")
	if err := app.Listen(":3000"); err != nil {
		log.Fatal("Server Listen Error: ", err)
	}
}
