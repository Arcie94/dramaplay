package models

import "gorm.io/gorm"

type Drama struct {
	BookID       string `gorm:"primaryKey" json:"bookId"`
	Judul        string `json:"judul"`
	Deskripsi    string `json:"deskripsi"`
	TotalEpisode string `json:"total_episode"`
	Cover        string `json:"cover"`
	Likes        string `json:"likes"`
	Genre        string `json:"genre"`
	IsFeatured   bool   `json:"isFeatured"`

	// Relations
	Episodes []Episode `gorm:"foreignKey:BookID;references:BookID" json:"episodes,omitempty"`
}

type Episode struct {
	ID           uint   `gorm:"primaryKey" json:"-"`
	BookID       string `gorm:"index" json:"bookId"`
	EpisodeIndex int    `json:"episode_index"`
	EpisodeLabel string `json:"episode_label"`

	// Stream Details (Hidden in detail view, shown in stream view via transformation)
	VideoURL string `json:"-"`
	Duration int    `json:"duration"`
}

// Responses structs for API compatibility

type SearchResponse struct {
	Status string  `json:"status"`
	Query  string  `json:"query,omitempty"`
	Data   []Drama `json:"data"`
}

type TrendingResponse struct {
	Status string  `json:"status"`
	Type   string  `json:"type"`
	Total  int64   `json:"total"`
	Data   []Drama `json:"data"`
}

type DetailResponse struct {
	Status                string    `json:"status"`
	BookID                string    `json:"bookId"`
	Judul                 string    `json:"judul"`
	Deskripsi             string    `json:"deskripsi"`
	Cover                 string    `json:"cover"`
	TotalEpisode          string    `json:"total_episode"`
	Likes                 string    `json:"likes"`
	JumlahEpisodeTersedia int       `json:"jumlah_episode_tersedia"`
	Episodes              []Episode `json:"episodes"`
}

type StreamResponse struct {
	Status string     `json:"status"`
	Data   StreamData `json:"data"`
}

type StreamData struct {
	BookID  string      `json:"bookId"`
	Chapter ChapterData `json:"chapter"`
}

type ChapterData struct {
	ID       string    `json:"id"`
	Index    int       `json:"index"`
	Duration int       `json:"duration"`
	Video    VideoData `json:"video"`
}

type VideoData struct {
	Mp4  string `json:"mp4"`
	M3u8 string `json:"m3u8"`
}

// MigrateDramas migrates the table
func MigrateDramas(db *gorm.DB) error {
	return db.AutoMigrate(&Drama{}, &Episode{})
}
