package handlers

import (
	"dramabang/database"
	"dramabang/models"
	"fmt"
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

// GetPublicSettings returns public settings (like Google Client ID) for the frontend
func GetPublicSettings(c *fiber.Ctx) error {
	var clientID models.Setting
	var gaID models.Setting
	// Only expose what's necessary
	database.DB.Where("key = ?", "google_client_id").First(&clientID)
	database.DB.Where("key = ?", "ga_measurement_id").First(&gaID)

	var siteLogo models.Setting
	// Explicitly find site_logo
	database.DB.Where("key = ?", "site_logo").First(&siteLogo)

	var siteFavicon models.Setting
	// Explicitly find site_favicon
	database.DB.Where("key = ?", "site_favicon").First(&siteFavicon)

	// Debug log
	fmt.Printf("DEBUG: GetPublicSettings - SiteLogo: %v\n", siteLogo)

	return c.JSON(fiber.Map{
		"status":            "success",
		"google_client_id":  clientID.Value,
		"ga_measurement_id": gaID.Value,
		"site_logo":         siteLogo.Value,
		"site_favicon":      siteFavicon.Value,
	})
}
