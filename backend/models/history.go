package models

import (
	"time"

	"gorm.io/gorm"
)

type UserHistory struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"userId" gorm:"index"` // Link to User
	BookID     string    `json:"bookId" gorm:"index"` // Link to Drama (BookID)
	EpisodeIdx int       `json:"episodeIdx"`          // 0-based index
	UpdatedAt  time.Time `json:"updatedAt"`

	// Relationship (Optional if you want to preload)
	Drama Drama `json:"drama" gorm:"foreignKey:BookID;references:BookID"`
}

func MigrateHistory(db *gorm.DB) {
	db.AutoMigrate(&UserHistory{})
}
