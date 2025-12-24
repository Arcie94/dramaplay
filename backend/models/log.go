package models

import (
	"time"

	"gorm.io/gorm"
)

type SystemLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Level     string    `json:"level"` // "INFO", "ERROR", "SUCCESS"
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func MigrateLogs(db *gorm.DB) error {
	return db.AutoMigrate(&SystemLog{})
}

func LogInfo(db *gorm.DB, message string) {
	db.Create(&SystemLog{
		Level:   "INFO",
		Message: message,
	})
}

func LogError(db *gorm.DB, message string) {
	db.Create(&SystemLog{
		Level:   "ERROR",
		Message: message,
	})
}

func LogSuccess(db *gorm.DB, message string) {
	db.Create(&SystemLog{
		Level:   "SUCCESS",
		Message: message,
	})
}
