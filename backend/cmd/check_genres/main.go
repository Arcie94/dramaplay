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

	type Result struct {
		Genre string
		Count int
	}

	var results []Result
	db.Model(&models.Drama{}).Select("genre, count(*) as count").Group("genre").Scan(&results)

	for _, res := range results {
		log.Printf("Genre: '%s' - Count: %d", res.Genre, res.Count)
	}
}
