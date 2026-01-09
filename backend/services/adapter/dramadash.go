package adapter

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type DramaDashProvider struct{}

const DramaDashAPI = "https://sapimu.au/dramadash" // Assuming standardized endpoints even on root path based on pattern

func NewDramaDashProvider() *DramaDashProvider {
	return &DramaDashProvider{}
}

func (p *DramaDashProvider) GetID() string {
	return "dramadash"
}

func (p *DramaDashProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *DramaDashProvider) fetch(targetURL string) ([]byte, error) {
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

func (p *DramaDashProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

type ddResponse struct {
	Status int `json:"status"`
	Data   struct {
		Banner []ddItem `json:"banner"`
	} `json:"data"`
}
type ddItem struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Poster string `json:"poster"`
	// Desc string `json:"desc"`
}

func (p *DramaDashProvider) GetTrending() ([]models.Drama, error) {
	// Use /home for Trending
	body, err := p.fetch(DramaDashAPI + "/home?lang=in")
	if err != nil {
		return nil, err
	}

	var raw ddResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	// Use Banner data for now
	for _, d := range raw.Data.Banner {
		dramas = append(dramas, models.Drama{
			BookID: "dramadash:" + strconv.Itoa(d.ID),
			Judul:  d.Name,
			Cover:  p.proxyImage(d.Poster),
		})
	}
	return dramas, nil
}

func (p *DramaDashProvider) GetLatest(page int) ([]models.Drama, error) {
	return p.GetTrending()
}

func (p *DramaDashProvider) Search(query string) ([]models.Drama, error) {
	return []models.Drama{}, nil
}

func (p *DramaDashProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	return nil, nil, fmt.Errorf("not implemented yet")
}

func (p *DramaDashProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	return nil, fmt.Errorf("not implemented yet")
}
