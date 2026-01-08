package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a registered user
type User struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Email             string         `gorm:"uniqueIndex;not null" json:"email"`
	Name              string         `json:"name"`
	Avatar            string         `json:"avatar" gorm:"type:text"`
	Provider          string         `json:"provider"`                 // google, apple, local
	Password          string         `json:"-"`                        // Stored as hash, ignored in JSON
	Role              string         `json:"role" gorm:"default:user"` // user, admin
	IsVerified        bool           `json:"is_verified" gorm:"default:false"`
	VerificationToken string         `json:"-"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// MigrateUsers migrates the table
func MigrateUsers(db *gorm.DB) error {
	return db.AutoMigrate(&User{})
}
