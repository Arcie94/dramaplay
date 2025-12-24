package handlers

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UploadFile handles file uploads for admin
func UploadFile(c *fiber.Ctx) error {
	// 1. Get the file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "No file uploaded"})
	}

	// 2. Validate file type (simple extension check)
	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid file type. Only images allowed."})
	}

	// 3. Generate unique filename
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), ext)

	// 4. Save to public/uploads
	// In Docker, this maps to /app/public/uploads which is mounted to ./uploads on host
	savePath := filepath.Join("public", "uploads", filename)

	if err := c.SaveFile(file, savePath); err != nil {
		fmt.Println("Upload Error:", err)
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to save file"})
	}

	// 5. Return the URL (relative to public)
	fileURL := fmt.Sprintf("/uploads/%s", filename)
	return c.JSON(fiber.Map{
		"status": "success",
		"url":    fileURL,
	})
}
