package main

import (
	"log"
	"strings"

	"dramabang/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Heuristic keyword mapping
var genreKeywords = map[string][]string{
	"Epic Fantasy & Mythology":   {"naga", "dewa", "legenda", "sakti", "myth", "fantasy", "abadi", "siluman", "langit", "bidadari"},
	"Martial Arts Action":        {"pendekar", "kungfu", "silat", "jurus", "pedang", "warrior", "fight", "jagoan", "master"},
	"Historical Romance":         {"kerajaan", "selir", "kaisar", "putri", "pangeran", "dynasty", "colossal", "takhta", "ratu"},
	"Palace Intrigue & Politics": {"intrik", "politik", "kudeta", "pengkhianatan", "hasutan", "skandal", "selingkuh", "politik"},
	"Modern Romance & CEO":       {"ceo", "bos", "presdir", "cinta", "nikah", "kontrak", "sekretaris", "miliarder", "kaya", "wealthy"},
	"School & Youth":             {"sekolah", "kampus", "mahasiswa", "sma", "kuliah", "youth", "remaja", "cinta pertama", "kelas"},
	"Mystery & Detective":        {"misteri", "detektif", "pembunuhan", "kasus", "investigasi", "rahasia", "hilang", "crime"},
	"E-Sports & Gaming":          {"game", "esport", "gamer", "kompetisi", "online", "avatar", "dunia maya"},
}

// Fallback genre if no keywords match
const DefaultGenre = "Modern Romance & CEO" // Most common in vertical dramas

func main() {
	dsn := "../../dramabang.db"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// AutoMigrate to ensure Genre column exists
	db.AutoMigrate(&models.Drama{})

	var dramas []models.Drama
	db.Find(&dramas)

	log.Printf("Classifying %d dramas...", len(dramas))

	for _, drama := range dramas {
		detectedGenre := classify(drama.Judul, drama.Deskripsi)

		// Only update if changed or empty
		if drama.Genre != detectedGenre {
			drama.Genre = detectedGenre
			db.Save(&drama)
			// log.Printf("Classified '%s' as '%s'", drama.Judul, detectedGenre)
		}
	}

	log.Println("Classification complete!")
}

func classify(title, desc string) string {
	text := strings.ToLower(title + " " + desc)

	// Check each genre
	for genre, keywords := range genreKeywords {
		for _, kw := range keywords {
			if strings.Contains(text, kw) {
				return genre
			}
		}
	}

	return DefaultGenre
}
