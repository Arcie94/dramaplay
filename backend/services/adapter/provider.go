package adapter

import (
	"dramabang/models"
)

// Provider maps a specific API source (e.g. Dramabox, Melolo)
type Provider interface {
	GetID() string // "dramabox", "melolo", "netshort"
	Search(query string) ([]models.Drama, error)
	GetTrending() ([]models.Drama, error)
	GetLatest(page int) ([]models.Drama, error)
	GetDetail(id string) (*models.Drama, []models.Episode, error)
	GetStream(id, epIndex string) (*models.StreamData, error)
	IsCompatibleID(id string) bool
}
