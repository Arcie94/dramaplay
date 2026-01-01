package handlers

import (
	"dramabang/database"
	"dramabang/models"
	"dramabang/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// VerifyGoogleToken receives the credential from frontend and validates it
func VerifyGoogleToken(c *fiber.Ctx) error {
	var input struct {
		Credential string `json:"credential"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	// 1. Decode JWT Payload (Without validation for MVP, but should verify signature in Prod)
	parts := strings.Split(input.Credential, ".")
	if len(parts) < 2 {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid token format"})
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Failed to decode token"})
	}

	var claims struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
		Sub     string `json:"sub"` // Google User ID
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Failed to parse claims"})
	}

	// 2. Find or Create User
	var user models.User
	// Use Unscoped to find soft-deleted users too
	result := database.DB.Unscoped().Where("email = ?", claims.Email).First(&user)

	if result.Error != nil {
		// Create new user
		user = models.User{
			Email:    claims.Email,
			Name:     claims.Name,
			Avatar:   claims.Picture,
			Provider: "google",
			Role:     "user",
		}
		if err := database.DB.Create(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to create user: " + err.Error()})
		}
	} else {
		// Update existing user info
		user.Name = claims.Name
		user.Avatar = claims.Picture

		// Restore user if soft-deleted
		if user.DeletedAt.Valid {
			database.DB.Model(&user).Update("deleted_at", nil)
			user.DeletedAt = gorm.DeletedAt{}
		}

		database.DB.Save(&user)
	}

	// 3. Return Session
	return c.JSON(fiber.Map{
		"status": "success",
		"token":  "session-token-" + time.Now().String(),
		"user": fiber.Map{
			"id":     user.ID,
			"email":  user.Email,
			"name":   user.Name,
			"avatar": user.Avatar,
			"role":   user.Role,
		},
	})
}

// LocalLogin handles email/password login and signup
func LocalLogin(c *fiber.Ctx) error {
	var input struct {
		Email               string `json:"email"`
		Password            string `json:"password"`
		Name                string `json:"name"` // Optional, for signup
		Mode                string `json:"mode"` // "login" or "signup"
		CFTurnstileResponse string `json:"cf_turnstile_response"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	// 1. Verify Turnstile (if configured)
	var secretKey models.Setting
	database.DB.Where("key = ?", "turnstile_secret_key").First(&secretKey)

	if secretKey.Value != "" {
		// if !verifyTurnstile(input.CFTurnstileResponse, secretKey.Value) {
		// 	return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Captcha validation failed"})
		// }
	}

	// Find User
	var user models.User
	result := database.DB.Where("email = ?", input.Email).First(&user)

	// --- SIGNUP MODE ---
	if input.Mode == "signup" {
		if result.Error == nil {
			// User already exists
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Email already registered. Please login."})
		}

		// Check if user exists (including soft-deleted)
		var existingUser models.User
		if err := database.DB.Unscoped().Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
			// User exists
			if existingUser.DeletedAt.Valid {
				// Soft-deleted user found -> Restore Account
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
				if err != nil {
					return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to hash password"})
				}

				// Update fields
				existingUser.Password = string(hashedPassword)
				existingUser.DeletedAt = gorm.DeletedAt{} // Restore
				if input.Name != "" {
					existingUser.Name = input.Name
				}
				// Reset provider to local if it was something else, to allow login
				existingUser.Provider = "local"

				if err := database.DB.Save(&existingUser).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to restore account"})
				}

				// Set 'user' to the restored user for the response
				user = existingUser
			} else {
				// Active user found (Should be caught by previous check, but just in case)
				return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Email already registered. Please login."})
			}
		} else {
			// Truly new user -> Create
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to hash password"})
			}

			if input.Name == "" {
				return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Username is required for signup"})
			}
			name := input.Name

			user = models.User{
				Email:    input.Email,
				Name:     name,
				Password: string(hashedPassword),
				Provider: "local",
				Role:     "user",
			}

			if err := database.DB.Create(&user).Error; err != nil {
				// Log the actual error for debugging
				fmt.Println("Signup Create Error:", err)
				return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to create user: " + err.Error()})
			}
		}

	} else {
		// --- LOGIN MODE (Default) ---
		if result.Error != nil {
			return c.Status(404).JSON(fiber.Map{"status": "error", "message": "User not found. Please sign up."})
		}

		// Check Provider
		if user.Provider == "google" {
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "This email is registered with Google. Please use Google Login."})
		}

		// --- MIGRATION FOR LEGACY USERS ---
		// If password is in database is empty (old account), set it now.
		if user.Password == "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to update legacy password"})
			}
			user.Password = string(hashedPassword)
			database.DB.Save(&user)
			fmt.Println("MIGRATED LEGACY USER:", user.Email) // Debug log
		} else {
			// Normal Verification
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
				return c.Status(401).JSON(fiber.Map{"status": "error", "message": "Invalid password"})
			}
		}

		// Update Last Seen? (Optional)
		database.DB.Save(&user)
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"token":  "session-token-" + time.Now().String(),
		"user": fiber.Map{
			"id":     user.ID,
			"email":  user.Email,
			"name":   user.Name,
			"avatar": user.Avatar,
		},
	})
}

// ForgotPassword handles generation of reset token and sending email
func ForgotPassword(c *fiber.Ctx) error {
	var input struct {
		Email string `json:"email"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// Do not reveal if email exists or not for security, but for UX we usually say "If email exists..."
		// For this app, simply returning success is safest, OR specific error if preferred by Client
		// Let's return success to avoid enumeration
		return c.JSON(fiber.Map{"status": "success", "message": "If your email is registered, you will receive a reset link."})
	}

	// Generate Token
	token := uuid.New().String()
	expiry := time.Now().Add(1 * time.Hour)

	// Create Reset Record
	resetToken := models.PasswordResetToken{
		Email:     user.Email,
		Token:     token,
		ExpiresAt: expiry,
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&resetToken).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Database error"})
	}

	// Send Email
	resetLink := fmt.Sprintf("https://dramaplay.online/reset-password/%s", token)
	emailBody := fmt.Sprintf(`
		<h3>Password Reset Request</h3>
		<p>Hi %s,</p>
		<p>You requested to reset your password. Click the link below to verify:</p>
		<p><a href="%s" style="padding: 10px 20px; background-color: #d90429; color: white; text-decoration: none; border-radius: 5px;">Reset Password</a></p>
		<p>Link expires in 1 hour.</p>
		<p>If you did not request this, please ignore this email.</p>
	`, user.Name, resetLink)

	// Send in Goroutine to not block response
	go func() {
		if err := utils.SendEmail([]string{user.Email}, "Reset Your Password - DramaPlay", emailBody); err != nil {
			fmt.Println("Failed to send email:", err)
		}
	}()

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Reset link sent to your email.",
	})
}

// ResetPassword handles the actual password update
func ResetPassword(c *fiber.Ctx) error {
	token := c.Params("token")
	var input struct {
		Password string `json:"password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	if len(input.Password) < 6 {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Password must be at least 6 characters"})
	}

	// Verify Token
	var resetToken models.PasswordResetToken
	if err := database.DB.Where("token = ? AND expires_at > ?", token, time.Now()).First(&resetToken).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid or expired token"})
	}

	// Update User Password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	if err := database.DB.Model(&models.User{}).Where("email = ?", resetToken.Email).Update("password", string(hashedPassword)).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to update password"})
	}

	// Delete used token (and all other tokens for this email to be clean)
	database.DB.Where("email = ?", resetToken.Email).Delete(&models.PasswordResetToken{})

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Password updated successfully. You can now login.",
	})
}
