package adapter

import (
	"dramabang/models"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type DramaWaveProvider struct{}

const DramaWaveAPI = "https://sapimu.au/dramawave/api/v1"

func NewDramaWaveProvider() *DramaWaveProvider {
	return &DramaWaveProvider{}
}

func (p *DramaWaveProvider) GetID() string {
	return "dramawave"
}

func (p *DramaWaveProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *DramaWaveProvider) fetch(targetURL string) ([]byte, error) {
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

func (p *DramaWaveProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// Stubs for currently unauthorized API

func (p *DramaWaveProvider) GetTrending() ([]models.Drama, error) {
	return []models.Drama{}, nil
}

func (p *DramaWaveProvider) GetLatest(page int) ([]models.Drama, error) {
	return []models.Drama{}, nil
}

func (p *DramaWaveProvider) Search(query string) ([]models.Drama, error) {
	return []models.Drama{}, nil
}

func (p *DramaWaveProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	return nil, nil, fmt.Errorf("dramawave api unauthorized (401)")
}

func (p *DramaWaveProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	return nil, fmt.Errorf("dramawave api unauthorized (401)")
}
