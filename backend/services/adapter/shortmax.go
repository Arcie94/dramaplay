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
}

type smEpisodeResponse struct {
	Data []smEpisode `json:"data"`
}
type smEpisode struct {
	ID      int  `json:"id"`
	Episode int  `json:"episode"`
	Locked  bool `json:"locked"`
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
	// Endpoint: /shortmax/api/v1/episodes/{id}?lang=id
	// Note: Metadata endpoint is broken (500), so we only fetch episodes.
	urlEpisodes := fmt.Sprintf("%s/episodes/%s?lang=id", ShortMaxAPI, id)
	body, err := p.fetch(urlEpisodes)
	if err != nil {
		return nil, nil, err
	}

	var raw smEpisodeResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	// Create partial drama object
	drama := models.Drama{
		BookID:       "shortmax:" + id,
		Judul:        "ShortMax Drama " + id, // Metadata unavailable
		Cover:        "",                     // Metadata unavailable
		Deskripsi:    "Metadata currently header unavailable from provider.",
		TotalEpisode: strconv.Itoa(len(raw.Data)),
	}

	var episodes []models.Episode
	for _, ep := range raw.Data {
		episodes = append(episodes, models.Episode{
			BookID:       "shortmax:" + id,
			EpisodeIndex: ep.Episode - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}

	return &drama, episodes, nil
}

func (p *ShortMaxProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	return nil, fmt.Errorf("stream api unavailable (500) from provider")
}
