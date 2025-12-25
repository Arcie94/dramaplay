package handlers

import (
	"dramabang/database"
	"dramabang/models"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// verifyTurnstile checks the token against Cloudflare's API
func verifyTurnstile(token, secret string) bool {
	if secret == "" {
		return true // Skip if no secret configured
	}
	if token == "" {
		return false // Fail if secret exists but no token provided
	}

	// Simple HTTP POST (without importing heavy net/http logic if possible, but here we need it)
	// We can use fiber's AcquireAgent or standard http
	agent := fiber.AcquireAgent()
	req := agent.Request()
	req.Header.SetMethod("POST")
	req.SetRequestURI("https://challenges.cloudflare.com/turnstile/v0/siteverify")

	// Form Data
	args := fiber.AcquireArgs()
	args.Set("secret", secret)
	args.Set("response", token)
	agent.Form(args)

	if err := agent.Parse(); err != nil {
		fmt.Println("Turnstile Agent Parse Error:", err)
		return false
	}

	code, body, errs := agent.Bytes()
	if len(errs) > 0 || code != 200 {
		fmt.Println("Turnstile Request Error:", errs)
		return false
	}

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Turnstile JSON Error:", err)
		return false
	}

	return result.Success
}

// AdminLogin handles authentication
func AdminLogin(c *fiber.Ctx) error {
	var input struct {
		Username            string `json:"username"`
		Password            string `json:"password"`
		CFTurnstileResponse string `json:"cf_turnstile_response"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	// 1. Verify Turnstile (if configured)
	// var secretKey models.Setting
	// database.DB.Where("key = ?", "turnstile_secret_key").First(&secretKey)

	// if secretKey.Value != "" {
	// 	if !verifyTurnstile(input.CFTurnstileResponse, secretKey.Value) {
	// 		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Captcha validation failed"})
	// 	}
	// }

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
	genre := c.Query("genre")
	sortBy := c.Query("sort")
	featured := c.Query("featured")

	if search != "" {
		query = query.Where("judul LIKE ?", "%"+search+"%")
	}
	if genre != "" {
		query = query.Where("genre LIKE ?", "%"+genre+"%")
	}
	if featured == "true" {
		query = query.Where("is_featured = ?", true)
	}

	// Count
	query.Count(&total)

	// Fetch Data
	dataQuery := database.DB.Model(&models.Drama{})
	if search != "" {
		dataQuery = dataQuery.Where("judul LIKE ?", "%"+search+"%")
	}
	if genre != "" {
		dataQuery = dataQuery.Where("genre LIKE ?", "%"+genre+"%")
	}
	if featured == "true" {
		dataQuery = dataQuery.Where("is_featured = ?", true)
	}

	// Sort
	switch sortBy {
	case "oldest":
		dataQuery = dataQuery.Order("book_id asc")
	case "title_asc":
		dataQuery = dataQuery.Order("judul asc")
	case "title_desc":
		dataQuery = dataQuery.Order("judul desc")
	default:
		dataQuery = dataQuery.Order("book_id desc") // newest
	}

	dataQuery.Limit(limit).Offset(offset).Find(&dramas)

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
	models.LogInfo(database.DB, "Ingest Process Triggered")
	// Run in background
	go func() {
		cmd := exec.Command("./ingest")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("Ingest Error:", err)
			models.LogError(database.DB, fmt.Sprintf("Ingest Failed: %v", err))
			return
		}
		// Log explicit success or output summary?
		// Truncate output if too long
		outStr := string(output)
		if len(outStr) > 100 {
			outStr = outStr[:100] + "..."
		}
		fmt.Println("Ingest Output:", string(output))
		models.LogSuccess(database.DB, "Ingest Finished: "+outStr)
	}()

	return c.JSON(fiber.Map{"status": "success", "message": "Ingest process started in background"})
}

// TriggerDedup runs the dedup script
func TriggerDedup(c *fiber.Ctx) error {
	models.LogInfo(database.DB, "Deduplication Process Triggered")
	// Run in background
	go func() {
		cmd := exec.Command("./dedup")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("Dedup Error:", err)
			models.LogError(database.DB, fmt.Sprintf("Dedup Failed: %v", err))
			return
		}
		outStr := string(output)
		if len(outStr) > 100 {
			outStr = outStr[:100] + "..."
		}
		fmt.Println("Dedup Output:", string(output))
		models.LogSuccess(database.DB, "Dedup Finished: "+outStr)
	}()

	return c.JSON(fiber.Map{"status": "success", "message": "Deduplication process started in background"})
}

// GetSystemLogs retrieves recent logs
func GetSystemLogs(c *fiber.Ctx) error {
	var logs []models.SystemLog
	if err := database.DB.Order("created_at desc").Limit(50).Find(&logs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Result error"})
	}
	return c.JSON(fiber.Map{"status": "success", "data": logs})
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
