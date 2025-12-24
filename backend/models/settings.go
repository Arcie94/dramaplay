package models

import "gorm.io/gorm"

// Setting represents a key-value configuration pair
type Setting struct {
	Key   string `gorm:"primaryKey" json:"key"`
	Value string `json:"value"`
}

// MigrateSettings migrates the table
func MigrateSettings(db *gorm.DB) error {
	return db.AutoMigrate(&Setting{})
}
