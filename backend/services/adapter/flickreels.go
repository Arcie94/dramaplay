package adapter

import (
	"dramabang/models"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type FlickReelsProvider struct{}

const FlickReelsAPI = "https://sapimu.au/flickreels/api/v1"

func NewFlickReelsProvider() *FlickReelsProvider {
	return &FlickReelsProvider{}
}

func (p *FlickReelsProvider) GetID() string {
	return "flickreels"
}

func (p *FlickReelsProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *FlickReelsProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Authorization", "Bearer "+SapimuToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (p *FlickReelsProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// Stubs for currently unauthorized API

func (p *FlickReelsProvider) GetTrending() ([]models.Drama, error) {
	// /rising returns 401, so we return empty or error.
	// Returning empty list is safer for Homepage to not crash.
	return []models.Drama{}, nil
}

func (p *FlickReelsProvider) GetLatest(page int) ([]models.Drama, error) {
	// /new returns 401
	return []models.Drama{}, nil
}

func (p *FlickReelsProvider) Search(query string) ([]models.Drama, error) {
	return []models.Drama{}, nil
}

func (p *FlickReelsProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	return nil, nil, fmt.Errorf("flickreels api unauthorized (401)")
}

func (p *FlickReelsProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	return nil, fmt.Errorf("flickreels api unauthorized (401)")
}
