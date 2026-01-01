package models

import (
	"time"

	"gorm.io/gorm"
)

type PasswordResetToken struct {
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"index;not null"`
	Token     string    `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

// MigrateTokens migrates the table
func MigrateTokens(db *gorm.DB) error {
	return db.AutoMigrate(&PasswordResetToken{})
}
