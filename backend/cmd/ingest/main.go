package main

import (
	"dramabang/database"
	"dramabang/models"
	"dramabang/services/adapter"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm/clause"
)

func main() {
	// Load .env if exists
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	database.Connect()
	log.Println("✅ Connected to database")

	// Ensure migrations
	models.MigrateDramas(database.DB)
	log.Println("✅ Database migrations complete")

	// Initialize providers
	providers := []adapter.Provider{
		adapter.NewMeloloProvider(),
		adapter.NewNetshortProvider(),
		// Dramabox is skipped due to API issues
	}

	totalDramas := 0

	for _, provider := range providers {
		providerName := provider.GetID()
		log.Printf("📥 Fetching from %s...", providerName)

		// Get trending dramas
		dramas, err := provider.GetTrending()
		if err != nil {
			log.Printf("⚠️  Error fetching from %s: %v", providerName, err)
			continue
		}

		if len(dramas) == 0 {
			log.Printf("⚠️  No dramas found from %s", providerName)
			continue
		}

		log.Printf("📦 Received %d dramas from %s", len(dramas), providerName)

		// Upsert dramas into database
		for _, drama := range dramas {
			if err := database.DB.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "book_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"judul", "cover", "deskripsi", "total_episode", "genre"}),
			}).Create(&drama).Error; err != nil {
				log.Printf("❌ Error saving drama %s: %v", drama.BookID, err)
			}
		}

		log.Printf("✅ Saved %d dramas from %s", len(dramas), providerName)
		totalDramas += len(dramas)

		// Also fetch latest (with pagination) for more content
		log.Printf("📥 Fetching latest from %s (page 1)...", providerName)
		latestDramas, err := provider.GetLatest(1)
		if err == nil && len(latestDramas) > 0 {
			for _, drama := range latestDramas {
				database.DB.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "book_id"}},
					DoUpdates: clause.AssignmentColumns([]string{"judul", "cover", "deskripsi", "total_episode", "genre"}),
				}).Create(&drama)
			}
			log.Printf("✅ Saved %d latest dramas from %s", len(latestDramas), providerName)
			totalDramas += len(latestDramas)
		}
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if totalDramas > 0 {
		log.Printf("🎉 Ingestion completed successfully!")
		log.Printf("📊 Total dramas inserted/updated: %d", totalDramas)
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		os.Exit(0)
	} else {
		log.Println("⚠️  No dramas were ingested. Check provider APIs.")
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		os.Exit(1)
	}
}
