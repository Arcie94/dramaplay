package adapter

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type HiShortProvider struct{}

const HiShortAPI = "https://sapimu.au/hishort/api"

func NewHiShortProvider() *HiShortProvider {
	return &HiShortProvider{}
}

func (p *HiShortProvider) GetID() string {
	return "hishort"
}

func (p *HiShortProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *HiShortProvider) fetch(targetURL string) ([]byte, error) {
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

func (p *HiShortProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

type hsResponse struct {
	Popular []hsItem `json:"popular"`
}
type hsItem struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Poster string `json:"poster"`
	Type   string `json:"type"`
}

func (p *HiShortProvider) GetTrending() ([]models.Drama, error) {
	// Use /home for Trending
	body, err := p.fetch(HiShortAPI + "/home?lang=in")
	if err != nil {
		return nil, err
	}

	var raw hsResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Popular {
		dramas = append(dramas, models.Drama{
			BookID: "hishort:" + d.Slug,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Poster),
		})
	}
	return dramas, nil
}

func (p *HiShortProvider) GetLatest(page int) ([]models.Drama, error) {
	return p.GetTrending()
}

func (p *HiShortProvider) Search(query string) ([]models.Drama, error) {
	return []models.Drama{}, nil
}

func (p *HiShortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	return nil, nil, fmt.Errorf("not implemented yet")
}

func (p *HiShortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	return nil, fmt.Errorf("not implemented yet")
}
