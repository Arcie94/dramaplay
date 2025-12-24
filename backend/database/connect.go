package database

import (
	"log"
	"os"

	"dramabang/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	driver := os.Getenv("DB_DRIVER")
	var dialector gorm.Dialector

	if driver == "postgres" {
		dsn := os.Getenv("DB_DSN")
		if dsn == "" {
			log.Fatal("DB_DSN environment variable is required for postgres driver")
		}
		dialector = postgres.Open(dsn)
		log.Println("Connecting to PostgreSQL...")
	} else {
		// Default to SQLite
		dsn := os.Getenv("DB_PATH")
		if dsn == "" {
			dsn = "dramabang.db"
		}
		dialector = sqlite.Open(dsn)
		log.Println("Connecting to SQLite...")
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
	}

	log.Println("Connected to Database successfully")

	log.Println("Running Auto Migrations...")
	db.AutoMigrate(&models.Drama{}, &models.Episode{}, &models.Setting{}, &models.User{}, &models.UserHistory{})

	DB = db
}
