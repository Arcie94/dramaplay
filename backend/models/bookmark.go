package models

import "gorm.io/gorm"

// Bookmark represents a user's saved drama
type Bookmark struct {
	gorm.Model
	UserID uint   `gorm:"index;not null" json:"userId"`
	BookID string `gorm:"index;not null" json:"bookId"`

	// Relationship
	User  User  `gorm:"foreignKey:UserID" json:"-"`
	Drama Drama `gorm:"foreignKey:BookID;references:BookID" json:"drama"`
}

func MigrateBookmarks(db *gorm.DB) error {
	return db.AutoMigrate(&Bookmark{})
}
