package main

import (
	"dramabang/models"
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 1. Load defaults or env
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found, using defaults/env vars")
	}

	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "../dramabang.db" // Default assumption
	}

	pgDSN := os.Getenv("POSTGRES_DSN")
	if pgDSN == "" {
		log.Fatal("POSTGRES_DSN is required (e.g., host=localhost user=dramabang password=dramabang dbname=dramabang port=5432 sslmode=disable)")
	}

	// 2. Connect via GORM
	log.Printf("Opening SQLite: %s\n", sqlitePath)
	srcDB, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal("Failed to open SQLite:", err)
	}

	log.Println("Opening PostgreSQL...")
	dstDB, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal("Failed to open PostgreSQL:", err)
	}

	// 3. Migrate Schema (Ensure tables exist)
	log.Println("Migrating Target Schema...")
	dstDB.AutoMigrate(&models.Drama{}, &models.Episode{}, &models.Setting{}, &models.User{}, &models.UserHistory{})

	// 4. Copy Data
	copyTable(srcDB, dstDB, &[]models.Setting{}, "Settings")
	copyTable(srcDB, dstDB, &[]models.User{}, "Users")
	copyTable(srcDB, dstDB, &[]models.Drama{}, "Dramas")
	copyTable(srcDB, dstDB, &[]models.Episode{}, "Episodes")
	copyTable(srcDB, dstDB, &[]models.UserHistory{}, "UserHistory") // Using UserHistory model

	log.Println("🎉 Migration Complete!")
}

func copyTable(src *gorm.DB, dst *gorm.DB, model interface{}, name string) {
	log.Printf("Copying %s...", name)

	// Fetch all from source
	if err := src.Find(model).Error; err != nil {
		log.Printf("Error reading %s: %v\n", name, err)
		return
	}

	// Check if slice is empty? Reflection is complex, let's trust Gorm batch create
	// But since model is a pointer to a slice, we can use Create directly.

	// Use clauses.OnConflict to avoid duplicates if re-run
	// But simple approach: Create.
	if err := dst.Create(model).Error; err != nil {
		log.Printf("Warning inserting %s (might strictly be duplicates): %v\n", name, err)
	} else {
		log.Printf("Successfully copied %s\n", name)
	}
}
