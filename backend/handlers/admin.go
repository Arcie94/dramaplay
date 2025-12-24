package handlers

import (
	"dramabang/database"
	"dramabang/models"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// AdminLogin handles authentication
func AdminLogin(c *fiber.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	if input.Username == "teddyayomi" && input.Password == "Arcie1994" {
		return c.JSON(fiber.Map{
			"status": "success",
			"token":  "admin-secret-token-123", // Simple static token for MVP
		})
	}

	return c.Status(401).JSON(fiber.Map{"status": "error", "message": "Invalid credentials"})
}

// GetAdminDramas lists dramas for the admin panel with pagination and search
func GetAdminDramas(c *fiber.Ctx) error {
	var dramas []models.Drama
	var total int64

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit
	search := c.Query("q")

	// Base query
	query := database.DB.Model(&models.Drama{})
	if search != "" {
		query = query.Where("judul LIKE ?", "%"+search+"%")
	}

	// Count
	query.Count(&total)

	// Fetch Data (Re-apply query conditions or use session)
	// Safest is to rebuild chain or use session, but let's just re-apply for clarity/safety.
	dataQuery := database.DB.Model(&models.Drama{})
	if search != "" {
		dataQuery = dataQuery.Where("judul LIKE ?", "%"+search+"%")
	}
	dataQuery.Limit(limit).Offset(offset).Order("book_id desc").Find(&dramas)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   dramas,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// UpdateDrama updates a drama's details
func UpdateDrama(c *fiber.Ctx) error {
	bookId := c.Params("id")
	var input models.Drama

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	var drama models.Drama
	if result := database.DB.First(&drama, "book_id = ?", bookId); result.Error != nil {
		return c.Status(404).JSON(fiber.Map{"status": "error", "message": "Drama not found"})
	}

	// Update fields
	drama.Judul = input.Judul
	drama.Cover = input.Cover
	drama.TotalEpisode = input.TotalEpisode
	drama.Deskripsi = input.Deskripsi
	drama.Genre = input.Genre

	database.DB.Save(&drama)

	return c.JSON(fiber.Map{"status": "success", "message": "Drama updated", "data": drama})
}

// DeleteDrama removes a drama from the database
func DeleteDrama(c *fiber.Ctx) error {
	bookId := c.Params("id")
	result := database.DB.Delete(&models.Drama{}, "book_id = ?", bookId)

	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to delete"})
	}

	return c.JSON(fiber.Map{"status": "success", "message": "Drama deleted"})
}

// TriggerIngest runs the ingest script
func TriggerIngest(c *fiber.Ctx) error {
	// Run in background
	go func() {
		cmd := exec.Command("go", "run", "cmd/ingest/main.go")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("Ingest Error:", err)
			return
		}
		fmt.Println("Ingest Output:", string(output))
	}()

	return c.JSON(fiber.Map{"status": "success", "message": "Ingest process started in background"})
}

// TriggerDedup runs the dedup script
func TriggerDedup(c *fiber.Ctx) error {
	// Run in background
	go func() {
		cmd := exec.Command("go", "run", "cmd/dedup/main.go")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("Dedup Error:", err)
			return
		}
		fmt.Println("Dedup Output:", string(output))
	}()

	return c.JSON(fiber.Map{"status": "success", "message": "Deduplication process started in background"})
}

// ToggleFeatured updates the IsFeatured status of a drama
func ToggleFeatured(c *fiber.Ctx) error {
	bookId := c.Params("id")
	var input struct {
		IsFeatured bool `json:"isFeatured"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	// Update target drama (Allow multiple featured items)
	if err := database.DB.Model(&models.Drama{}).Where("book_id = ?", bookId).Update("is_featured", input.IsFeatured).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to update feature status"})
	}

	return c.JSON(fiber.Map{"status": "success", "message": "Featured status updated"})
}
