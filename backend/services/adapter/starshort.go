package adapter

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type StarshortProvider struct{}

const StarshortAPI = "https://dramabos.asia/api/starshort/api/v1"

func NewStarshortProvider() *StarshortProvider {
	return &StarshortProvider{}
}

func (p *StarshortProvider) GetID() string {
	return "starshort"
}

func (p *StarshortProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *StarshortProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	// Browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
	req.Header.Set("Referer", "https://dramabos.asia/")
	req.Header.Set("Origin", "https://dramabos.asia")

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

func (p *StarshortProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Internal Models ---

type sHomeResponse struct {
	// Data is a map of category names to list of dramas
	Data map[string][]sDrama `json:"data"`
}

type sSearchResponse struct {
	// Search usually returns just a list of dramas in 'data' or similar structure
	// Let's assume standard response wrapper or list
	Data []sDrama `json:"data"`
}

type sDrama struct {
	ID       int      `json:"id"` // ID is int in JSON
	FakeID   string   `json:"fakeId"`
	Title    string   `json:"title"`
	Cover    string   `json:"cover"`
	Episodes int      `json:"episodes"`
	Views    int      `json:"views"`
	Tags     []string `json:"tags"`
	Summary  string   `json:"summary"`
}

type sDetailResponse struct {
	Data sDetailData `json:"data"`
}

type sDetailData struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	Cover         string `json:"cover"`
	TotalEpisodes int    `json:"episodes"`
	// API might differ slightly, inferred from 'episodes' in Home
	Tags []string `json:"tags"`
}

// Episode list response ?
// User URL: /api/v1/episodes/myn
type sEpisodeListResponse struct {
	Data []sEpisode `json:"data"`
}

type sEpisode struct {
	ID      int `json:"id"`
	Episode int `json:"name"`   // Often "name" represents index or number
	Locked  int `json:"locked"` // 1 or 0
}

// Stream response
// User URL: /api/v1/play/myn?ep=1
type sPlayResponse struct {
	Data struct {
		URL string `json:"url"` // Similar to Melolo
	} `json:"data"`
}

// --- Implementation ---

func (p *StarshortProvider) GetTrending() ([]models.Drama, error) {
	// Use /home?lang=4 for Trending
	body, err := p.fetch(StarshortAPI + "/home?lang=4")
	if err != nil {
		return nil, err
	}

	var raw sHomeResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	// Flatten all categories
	for _, list := range raw.Data {
		for _, d := range list {
			dramas = append(dramas, models.Drama{
				BookID:       "starshort:" + strconv.Itoa(d.ID),
				Judul:        d.Title,
				Cover:        p.proxyImage(d.Cover),
				Deskripsi:    d.Summary,
				TotalEpisode: strconv.Itoa(d.Episodes),
				Genre:        strings.Join(d.Tags, ", "),
			})
		}
	}
	return dramas, nil
}

func (p *StarshortProvider) GetLatest(page int) ([]models.Drama, error) {
	// Use same as trending for now as no explicit latest pagination in user provided endpoints
	// Or maybe one of the categories in Home is "New"?
	return p.GetTrending()
}

func (p *StarshortProvider) Search(query string) ([]models.Drama, error) {
	// Endpoint: /search?q=cinta&lang=4
	urlSearch := fmt.Sprintf("%s/search?q=%s&lang=4", StarshortAPI, url.QueryEscape(query))
	body, err := p.fetch(urlSearch)
	if err != nil {
		return nil, err
	}

	// Search response might be wrapped in {data: []} or just []
	// Let's assume {data: []} first based on Home
	var raw sSearchResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Data {
		dramas = append(dramas, models.Drama{
			BookID:       "starshort:" + strconv.Itoa(d.ID),
			Judul:        d.Title,
			Cover:        p.proxyImage(d.Cover),
			Deskripsi:    d.Summary,
			TotalEpisode: strconv.Itoa(d.Episodes),
			Genre:        strings.Join(d.Tags, ", "),
		})
	}
	return dramas, nil
}

func (p *StarshortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// 1. Fetch Detail Info: /drama/{id}?lang=4
	urlDetail := fmt.Sprintf("%s/drama/%s?lang=4", StarshortAPI, id)
	body, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}

	var raw sDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}
	d := raw.Data

	drama := models.Drama{
		BookID:       "starshort:" + strconv.Itoa(d.ID),
		Judul:        d.Title,
		Cover:        p.proxyImage(d.Cover),
		Deskripsi:    d.Summary,
		TotalEpisode: strconv.Itoa(d.TotalEpisodes),
		Genre:        strings.Join(d.Tags, ", "),
	}

	// 2. Fetch Episodes List: /episodes/{id}?lang=4
	urlEp := fmt.Sprintf("%s/episodes/%s?lang=4", StarshortAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return &drama, nil, err
	}

	// Episode response likely {data: [...]}
	var rawEp sEpisodeListResponse
	if err := json.Unmarshal(bodyEp, &rawEp); err != nil {
		// Fallback: maybe just []sEpisode?
		// But let's assume consistent Data wrapper
		return &drama, nil, err
	}

	var episodes []models.Episode
	for _, ep := range rawEp.Data {
		episodes = append(episodes, models.Episode{
			BookID:       "starshort:" + id,
			EpisodeIndex: ep.Episode - 1, // Assuming 'name' or 'episode' is 1-based index
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}

	return &drama, episodes, nil
}

func (p *StarshortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// Endpoint: /play/{id}?ep={ep}&lang=4
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1 // API uses 1-based param based on user url example: ep=1

	urlPlay := fmt.Sprintf("%s/play/%s?ep=%d&lang=4", StarshortAPI, id, epNum)
	body, err := p.fetch(urlPlay)
	if err != nil {
		return nil, err
	}

	var raw sPlayResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	if raw.Data.URL == "" {
		return nil, fmt.Errorf("no video url found")
	}

	return &models.StreamData{
		BookID: "starshort:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.Data.URL,
			},
		},
	}, nil
}
