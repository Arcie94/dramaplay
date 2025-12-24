package handlers

import (
	"dramabang/database"
	"dramabang/models"
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// GetAdminUsers returns a paginated list of users with their last watch history
func GetAdminUsers(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	search := c.Query("search", "")

	var users []models.User
	var total int64

	// Base Query
	query := database.DB.Model(&models.User{})

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("name LIKE ? OR email LIKE ?", searchTerm, searchTerm)
	}

	// Count total users (filtered)
	if err := query.Count(&total).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Database error"})
	}

	// Fetch paginated users (filtered)
	if err := query.Limit(limit).Offset(offset).Order("created_at desc").Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Database error"})
	}

	// Enrich with Last Watch History
	var userList []map[string]interface{}
	for _, user := range users {
		var lastWatch models.UserHistory
		// Find latest history for this user
		err := database.DB.Preload("Drama").
			Where("user_id = ?", user.ID).
			Order("updated_at desc").
			First(&lastWatch).Error

		userData := map[string]interface{}{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"role":       user.Role,
			"avatar":     user.Avatar, // Include Avatar
			"provider":   user.Provider,
			"created_at": user.CreatedAt,
			"last_watch": nil, // Default null
		}

		// If found and has valid drama
		if err == nil && lastWatch.ID != 0 {
			userData["last_watch"] = map[string]interface{}{
				"drama_title": lastWatch.Drama.Judul,
				"episode":     lastWatch.EpisodeIdx + 1, // Display as 1-based
				"updated_at":  lastWatch.UpdatedAt,
			}
		}

		userList = append(userList, userData)
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   userList,
		"page":   page,
		"limit":  limit,
		"total":  total,
		"pages":  math.Ceil(float64(total) / float64(limit)),
	})
}

// DeleteUser soft-deletes a user
func DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "User ID is required"})
	}

	result := database.DB.Delete(&models.User{}, id)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to delete user"})
	}

	if result.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"status": "error", "message": "User not found"})
	}

	return c.JSON(fiber.Map{"status": "success", "message": "User deleted successfully"})
}

// UpdateUserProfile allows users to update their own profile (Name, Avatar)
func UpdateUserProfile(c *fiber.Ctx) error {
	type UpdateRequest struct {
		ID     uint   `json:"id"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	var user models.User
	if err := database.DB.First(&user, req.ID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"status": "error", "message": "User not found"})
	}

	// Update fields
	if req.Name != "" {
		user.Name = req.Name
	}
	// Always update avatar if sent (even if empty string to remove?)
	// For now, let's assume partial updates.
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	database.DB.Save(&user)

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Profile updated",
		"data":    user,
	})
}
