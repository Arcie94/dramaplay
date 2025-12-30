package handlers

import (
	"dramabang/database"
	"dramabang/models"

	"github.com/gofiber/fiber/v2"
)

// AddBookmark adds a drama to user's list
func AddBookmark(c *fiber.Ctx) error {
	type Request struct {
		UserID uint   `json:"userId"`
		BookID string `json:"bookId"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	// 1. Check if already exists
	var count int64
	database.DB.Model(&models.Bookmark{}).Where("user_id = ? AND book_id = ?", req.UserID, req.BookID).Count(&count)
	if count > 0 {
		return c.JSON(fiber.Map{"status": "success", "message": "Already in list"})
	}

	// 2. Ensure Drama exists (Lazy Ingest)
	// Check if drama is in local DB
	var drama models.Drama
	if err := database.DB.Where("book_id = ?", req.BookID).First(&drama).Error; err != nil {
		// Not found, fetch from Universal Adapter
		fetchedDrama, _, err := AdapterManager.GetDetail(req.BookID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"status": "error", "message": "Drama not found"})
		}
		// Save to DB
		database.DB.Save(fetchedDrama)
	}

	// 3. Create Bookmark
	bookmark := models.Bookmark{
		UserID: req.UserID,
		BookID: req.BookID,
	}

	if err := database.DB.Create(&bookmark).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to save bookmark"})
	}

	return c.JSON(fiber.Map{"status": "success"})
}

// RemoveBookmark removes a drama from user's list
func RemoveBookmark(c *fiber.Ctx) error {
	userId := c.Query("userId") // We trust frontend to send ID if not using strict JWT yet
	bookId := c.Params("bookId")

	if userId == "" || bookId == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Missing params"})
	}

	result := database.DB.Where("user_id = ? AND book_id = ?", userId, bookId).Delete(&models.Bookmark{})
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Database error"})
	}

	return c.JSON(fiber.Map{"status": "success"})
}

// GetBookmarks fetches user's list
func GetBookmarks(c *fiber.Ctx) error {
	userId := c.Query("userId")
	if userId == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Missing userId"})
	}

	var bookmarks []models.Bookmark
	// Preload Drama
	if err := database.DB.Preload("Drama").Where("user_id = ?", userId).Order("created_at desc").Find(&bookmarks).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Database error"})
	}

	// Extract just the dramas for cleaner frontend consumption?
	// Or keep structure: { data: [{ bookId:..., drama: {...} }] }
	// Let's keep structure but ensure Drama is populated

	// Lazy Ingest check (redundant if AddBookmark always ensures uniqueness, but safe for old data/migrations)
	// We'll skip lazy ingest on GET for speed, relying on Add/History to populate Drama table.

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   bookmarks,
	})
}

// CheckBookmark checks if a specific drama is in user's list
func CheckBookmark(c *fiber.Ctx) error {
	userId := c.Query("userId")
	bookId := c.Params("bookId")

	if userId == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Missing userId"})
	}

	var count int64
	database.DB.Model(&models.Bookmark{}).Where("user_id = ? AND book_id = ?", userId, bookId).Count(&count)

	return c.JSON(fiber.Map{
		"status": "success",
		"exists": count > 0,
	})
}
