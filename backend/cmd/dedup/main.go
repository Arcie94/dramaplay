package main

import (
	"dramabang/models"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	dsn := "../../dramabang.db"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Find duplicates
	type Result struct {
		Judul string
		Count int
	}

	var results []Result
	db.Model(&models.Drama{}).Select("judul, count(*) as count").Group("judul").Having("count > 1").Find(&results)

	log.Printf("Found %d titles with duplicates", len(results))

	for _, res := range results {
		var dramas []models.Drama
		db.Where("judul = ?", res.Judul).Order("book_id desc").Find(&dramas)

		// Keep the first one (latest book_id), delete the rest
		if len(dramas) > 1 {
			log.Printf("Deduplicating: %s (Found %d)", res.Judul, len(dramas))
			for i := 1; i < len(dramas); i++ {
				log.Printf(" - Deleting duplicate: %s (ID: %s)", dramas[i].Judul, dramas[i].BookID)
				db.Delete(&dramas[i])
			}
		}
	}

	log.Println("Deduplication complete.")
}
