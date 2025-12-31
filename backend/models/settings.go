package models

import "gorm.io/gorm"

// Setting represents a key-value configuration pair
type Setting struct {
	Key   string `gorm:"primaryKey" json:"key"`
	Value string `json:"value"`
}

// MigrateSettings migrates the table
func MigrateSettings(db *gorm.DB) error {
	err := db.AutoMigrate(&Setting{})
	if err != nil {
		return err
	}

	// Seed Google Client ID if missing
	var s Setting
	if db.Where("key = ?", "google_client_id").First(&s).Error != nil {
		db.Create(&Setting{
			Key:   "google_client_id",
			Value: "948421850128-kh10okq8tvc2rnl6vd4d460s1r3r7vir.apps.googleusercontent.com",
		})
	}

	// Seed Site Logo if missing
	if db.Where("key = ?", "site_logo").First(&s).Error != nil {
		db.Create(&Setting{
			Key:   "site_logo",
			Value: "/logo-404.png",
		})
	}
	return nil
}
