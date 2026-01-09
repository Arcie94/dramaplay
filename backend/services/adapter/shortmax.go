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

type ShortMaxProvider struct{}

const ShortMaxAPI = "https://sapimu.au/shortmax/api/v1"

func NewShortMaxProvider() *ShortMaxProvider {
	return &ShortMaxProvider{}
}

func (p *ShortMaxProvider) GetID() string {
	return "shortmax"
}

func (p *ShortMaxProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *ShortMaxProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
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

func (p *ShortMaxProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// Reusing same structs with `sm` prefix for safety
type smResponse struct {
	Data []smItem `json:"data"`
}
type smItem struct {
	ID    int    `json:"id"`
	Code  int    `json:"code"`
	Name  string `json:"name"`
	Cover string `json:"cover"`
	// Summary string `json:"summary"`
}

// smDetail/smEpisode likely differ too, but concentrating on Home for now

func (p *ShortMaxProvider) GetTrending() ([]models.Drama, error) {
	// Use /home for Trending
	body, err := p.fetch(ShortMaxAPI + "/home?lang=id")
	if err != nil {
		return nil, err
	}

	var raw smResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Data {
		dramas = append(dramas, models.Drama{
			BookID: "shortmax:" + strconv.Itoa(d.ID),
			Judul:  d.Name,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *ShortMaxProvider) GetLatest(page int) ([]models.Drama, error) {
	return p.GetTrending()
}

func (p *ShortMaxProvider) Search(query string) ([]models.Drama, error) {
	return []models.Drama{}, nil
}

func (p *ShortMaxProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	return nil, nil, fmt.Errorf("not implemented yet")
}

func (p *ShortMaxProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	return nil, fmt.Errorf("not implemented yet")
}
