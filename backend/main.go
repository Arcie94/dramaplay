package main

import (
	"dramabang/database"
	"dramabang/handlers"
	"dramabang/models"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	app := fiber.New()

	// Security Middleware
	app.Use(helmet.New()) // XSS, Clickjacking, etc.

	// Rate Limiting (100 reqs / min)
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() // Limit by IP
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  "error",
				"message": "Too many requests, please try again later.",
			})
		},
	}))

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000,http://localhost:4321,https://dramaplay.online,https://www.dramaplay.online",
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

	// FORCE MANUAL MIGRATION as Fallback
	// Ensure table exists for postgres (since AutoMigrate is sometimes flaky on new tables in live envs)
	database.DB.Exec(`
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL,
			token TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_email ON password_reset_tokens(email);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token);
	`)

	log.Println("Starting server on :3000...")

	// Routes
	api := app.Group("/api")

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
	api.Get("/auth/verify", handlers.VerifyEmail)        // New Verification Endpoint
	api.Put("/user/profile", handlers.UpdateUserProfile) // New Profile Update
	api.Put("/user/password", handlers.UpdatePassword)   // New Password Change
	api.Post("/forgot-password", handlers.ForgotPassword)
	api.Post("/reset-password/:token", handlers.ResetPassword)

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
	api.Get("/history/check", handlers.CheckHistory)

	// Admin Login (Public)
	api.Post("/admin/login", handlers.AdminLogin)

	// Admin config (Protected)
	admin := api.Group("/admin")
	admin.Use(func(c *fiber.Ctx) error {
		// Get token from header
		token := c.Get("Authorization")
		secret := os.Getenv("ADMIN_SECRET")
		if secret == "" {
			// Fallback if env not set (Safety net, but should be set)
			log.Println("WARNING: ADMIN_SECRET not set in env")
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Server Config Error"})
		}

		if token != secret {
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
	admin.Post("/settings", handlers.UpdateSettings)
	admin.Post("/upload", handlers.UploadFile) // Secured under /admin group

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
