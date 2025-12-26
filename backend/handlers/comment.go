package handlers

import (
	"dramabang/database"
	"dramabang/models"

	"github.com/gofiber/fiber/v2"
)

// GetComments fetches comments for a specific drama/book including user info
func GetComments(c *fiber.Ctx) error {
	bookID := c.Params("bookId")
	if bookID == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Book ID is required"})
	}

	var comments []models.Comment
	// Preload User to get name and avatar
	if err := database.DB.Preload("User").Where("book_id = ?", bookID).Order("created_at desc").Find(&comments).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Database error"})
	}

	// Transform data for easier frontend consumption
	var commentList []map[string]interface{}
	for _, comment := range comments {
		commentList = append(commentList, map[string]interface{}{
			"id":         comment.ID,
			"user_id":    comment.UserID, // Explicitly return user_id for owner check
			"content":    comment.Content,
			"created_at": comment.CreatedAt,
			"user": map[string]interface{}{
				"id":     comment.User.ID,
				"name":   comment.User.Name,
				"avatar": comment.User.Avatar,
			},
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   commentList,
	})
}

// PostComment saves a new comment
func PostComment(c *fiber.Ctx) error {
	type CreateCommentRequest struct {
		BookID  string `json:"book_id"`
		UserID  uint   `json:"user_id"` // In real app, get from Context/Token
		Content string `json:"content"`
	}

	var req CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	if req.BookID == "" || req.Content == "" || req.UserID == 0 {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Missing fields"})
	}

	// Verify User exists
	var user models.User
	if err := database.DB.First(&user, req.UserID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"status": "error", "message": "User not found"})
	}

	comment := models.Comment{
		BookID:  req.BookID,
		UserID:  req.UserID,
		Content: req.Content,
	}

	if err := database.DB.Create(&comment).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to save comment"})
	}

	// Return the FULL comment structure with user info so frontend can prepend it immediately
	response := map[string]interface{}{
		"id":         comment.ID,
		"user_id":    comment.UserID,
		"content":    comment.Content,
		"created_at": comment.CreatedAt, // usually time.Now()
		"user": map[string]interface{}{
			"id":     user.ID,
			"name":   user.Name,
			"avatar": user.Avatar,
		},
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Comment posted",
		"data":    response,
	})
}

// UpdateComment modifies an existing comment
func UpdateComment(c *fiber.Ctx) error {
	id := c.Params("id")
	type UpdateCommentRequest struct {
		Content string `json:"content"`
	}

	var req UpdateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	if req.Content == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Content is required"})
	}

	var comment models.Comment
	if err := database.DB.First(&comment, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"status": "error", "message": "Comment not found"})
	}

	// Ideally execute owner check here using JWT token from context
	// e.g., if comment.UserID != c.Locals("user_id") { return 403 }
	// For now, relying on frontend check + open logic as per current setup

	comment.Content = req.Content
	if err := database.DB.Save(&comment).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to update comment"})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Comment updated",
		"data":    comment,
	})
}

// DeleteComment removes a comment
func DeleteComment(c *fiber.Ctx) error {
	id := c.Params("id")

	var comment models.Comment
	if err := database.DB.First(&comment, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"status": "error", "message": "Comment not found"})
	}

	// Ideally execute owner check here

	if err := database.DB.Delete(&comment).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to delete comment"})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Comment deleted",
	})
}
