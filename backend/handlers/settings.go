package handlers

import (
	"dramabang/database"
	"dramabang/models"
	"os"

	"github.com/gofiber/fiber/v2"
)

// GetSettings retrieves all settings or a specific key
func GetSettings(c *fiber.Ctx) error {
	var settings []models.Setting
	database.DB.Find(&settings)

	// Convert to map for easier frontend consumption
	settingMap := make(map[string]string)
	for _, s := range settings {
		settingMap[s.Key] = s.Value
	}

	return c.JSON(fiber.Map{"status": "success", "data": settingMap})
}

// UpdateSettings updates a specific setting
func UpdateSettings(c *fiber.Ctx) error {
	var input struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	var setting models.Setting
	// Check if exists
	if err := database.DB.Where("key = ?", input.Key).First(&setting).Error; err != nil {
		// New setting
		setting = models.Setting{Key: input.Key, Value: input.Value}
		if err := database.DB.Create(&setting).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to create setting"})
		}
	} else {
		// Update existing
		setting.Value = input.Value
		if err := database.DB.Save(&setting).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to update setting"})
		}
	}

	// SPECIAL: If tunnel token, write to file sharing
	if input.Key == "cloudflare_tunnel_token" {
		if err := os.MkdirAll("data", 0755); err == nil {
			os.WriteFile("data/tunnel_token", []byte(input.Value), 0644)
		}
	}

	return c.JSON(fiber.Map{"status": "success", "data": setting})
}

// UpdateSettingsBatch updates multiple settings at once
func UpdateSettingsBatch(c *fiber.Ctx) error {
	var input map[string]string
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	tx := database.DB.Begin()

	for k, v := range input {
		var setting models.Setting
		if err := tx.Where("key = ?", k).First(&setting).Error; err != nil {
			// Create
			setting = models.Setting{Key: k, Value: v}
			if err := tx.Create(&setting).Error; err != nil {
				tx.Rollback()
				return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to create " + k})
			}
		} else {
			// Update
			setting.Value = v
			if err := tx.Save(&setting).Error; err != nil {
				tx.Rollback()
				return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to update " + k})
			}
		}

		// SPECIAL: Tunnel Token File
		if k == "cloudflare_tunnel_token" {
			if err := os.MkdirAll("data", 0755); err == nil {
				os.WriteFile("data/tunnel_token", []byte(v), 0644)
			}
		}
	}

	tx.Commit()
	return c.JSON(fiber.Map{"status": "success", "message": "All settings updated"})
}

// GetPublicSettings returns public settings (like Google Client ID) for the frontend
func GetPublicSettings(c *fiber.Ctx) error {
	var settings []models.Setting
	database.DB.Find(&settings)

	// Convert to map
	sMap := make(map[string]string)
	for _, s := range settings {
		sMap[s.Key] = s.Value
	}

	return c.JSON(fiber.Map{
		"status":             "success",
		"google_client_id":   sMap["google_client_id"],
		"ga_measurement_id":  sMap["ga_measurement_id"],
		"site_logo":          sMap["site_logo"],
		"site_favicon":       sMap["site_favicon"],
		"turnstile_site_key": sMap["turnstile_site_key"],
		"social_image":       sMap["social_image"],
	})
}
