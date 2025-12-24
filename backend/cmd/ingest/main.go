package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"dramabang/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ScraperURL = "http://localhost:3001/api/latest"
	StartPage  = 51
	EndPage    = 500 // Expanded to cover full library
)

type APIResponse struct {
	Status string         `json:"status"`
	Data   []models.Drama `json:"data"`
}

func main() {
	// Connect to Database
	dsn := "../../dramabang.db" // Relative path from cmd/ingest/
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	log.Println("Connected to Database")

	// Ensure Migrations
	db.AutoMigrate(&models.Drama{}, &models.Episode{})

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for i := StartPage; i <= EndPage; i++ {
		log.Printf("Fetching Page %d...", i)
		url := fmt.Sprintf("%s?page=%d", ScraperURL, i)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("Error fetching page %d: %v", i, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("Status code %d for page %d", resp.StatusCode, i)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			log.Printf("Error decoding JSON page %d: %v", i, err)
			continue
		}

		if len(apiResp.Data) == 0 {
			log.Println("No more data found. Stopping.")
			break
		}

		// Batch Upsert
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "book_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"judul", "cover", "total_episode"}),
		}).Create(&apiResp.Data).Error; err != nil {
			log.Printf("Error saving page %d: %v", i, err)
		} else {
			log.Printf("Successfully saved %d dramas from page %d", len(apiResp.Data), i)
		}

		// Be polite to the scraper/server
		time.Sleep(500 * time.Millisecond)
	}

	log.Println("Ingestion Complete!")
}
