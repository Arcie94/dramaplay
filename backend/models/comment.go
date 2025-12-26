package models

import (
	"gorm.io/gorm"
)

type Comment struct {
	gorm.Model
	UserID  uint   `json:"user_id"`
	User    User   `json:"user" gorm:"foreignKey:UserID"`
	BookID  string `json:"book_id" gorm:"index"` // References Drama ID
	Content string `json:"content" gorm:"type:text"`
}
