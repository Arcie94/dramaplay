package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

	// Seed Admin Credentials if missing
	if db.Where("key = ?", "admin_username").First(&s).Error != nil {
		db.Create(&Setting{
			Key:   "admin_username",
			Value: "teddyayomi",
		})
	}

	if db.Where("key = ?", "admin_password").First(&s).Error != nil {
		// Arc!e1994 (Hashed)
		// We use a pre-calculated hash or calculate it here.
		// Since we don't want to import bcrypt just for this if we can avoid it,
		// but we need it for verification anyway. Let's assume we import bcrypt.
		// Actually, let's use a hardcoded hash for "Arcie1994" to avoid import cycles or bloat if not needed here.
		// Wait, models shouldn't depend on utils if utils depend on models.
		// It's safer to import "golang.org/x/crypto/bcrypt" here.

		// "Arcie1994" hash cost 10: $2a$10$7X... (Generated now for safety)
		// For simplicity and correctness, I will use the code to generate it.
		// I will need to add the import above.
		hash, _ := bcrypt.GenerateFromPassword([]byte("Arcie1994"), bcrypt.DefaultCost)
		db.Create(&Setting{
			Key:   "admin_password",
			Value: string(hash),
		})
	}

	// Seed App Version if missing
	if db.Where("key = ?", "app_version").First(&s).Error != nil {
		db.Create(&Setting{
			Key:   "app_version",
			Value: "v1.2.0-beta",
		})
	}

	return nil
}
