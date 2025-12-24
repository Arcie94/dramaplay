package database

import (
	"log"

	"dramabang/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	// SQLite file path
	dsn := "dramabang.db"

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
	}

	log.Println("Connected to SQLite Database successfully")

	log.Println("Running Auto Migrations...")
	db.AutoMigrate(&models.Drama{}, &models.Episode{})

	DB = db
}
