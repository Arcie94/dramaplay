package adapter

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type DramaWaveProvider struct {
	client *http.Client
}

const DramaWaveAPI = "https://dramabos.asia/api/dramawave/api/v1"

func NewDramaWaveProvider() *DramaWaveProvider {
	return &DramaWaveProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://dramabos.asia/")

	resp, err := p.client.Do(req)
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

// --- Models ---
// Structure based on Debug: data.items[].items[]
type dwFeedResponse struct {
	Code int `json:"code"`
	Data struct {
		Items []dwModule `json:"items"`
	} `json:"data"`
}

type dwModule struct {
	Type  string    `json:"type"`
	Items []dwDrama `json:"items"`
}

type dwDrama struct {
	Key     string `json:"key"` // ID is string
	Title   string `json:"title"`
	Cover   string `json:"cover"`
	Desc    string `json:"desc"`
	EpCount int    `json:"episode_count"`
}

// Detail: /dramas/{id}
type dwDetailResp struct {
	Data dwDetailData `json:"data"`
}
type dwDetailData struct {
	ID           string `json:"key"` // assume key matches
	Title        string `json:"title"`
	Cover        string `json:"cover"`
	Introduction string `json:"introduction"` // or desc?
	EpisodeCount int    `json:"episode_count"`
	Episodes     []dwEp `json:"episodes"`
	// Or maybe episode info is nested differently.
	// But let's assume standard based on user endpoint play/1 implies index 1.
}
type dwEp struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	// PlayUrl string `json:"play_url"` // usually not exposed here
}

// Stream: /dramas/{id}/play/{ep}
type dwStreamResp struct {
	Data struct {
		PlayUrl string `json:"play_url"` // Guessing key
		Url     string `json:"url"`      // Alternate guess
		M3u8    string `json:"m3u8"`     // Alternate
	} `json:"data"`
}

// --- Implementation ---

func (p *DramaWaveProvider) GetTrending() ([]models.Drama, error) {
	// /feed/popular?lang=id
	url := DramaWaveAPI + "/feed/popular?lang=id"
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp dwFeedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	// Flatten modules
	for _, mod := range resp.Data.Items {
		for _, d := range mod.Items {
			// Avoid duplicates if same drama in multiple modules?
			// Simple append for now
			dramas = append(dramas, models.Drama{
				BookID:       "dramawave:" + d.Key,
				Judul:        d.Title,
				Cover:        p.proxyImage(d.Cover),
				Deskripsi:    d.Desc,
				TotalEpisode: strconv.Itoa(d.EpCount),
			})
		}
	}
	return dramas, nil
}

func (p *DramaWaveProvider) GetLatest(page int) ([]models.Drama, error) {
	// /feed/new?lang=id
	// Using Trending logic but different endpoint
	url := DramaWaveAPI + "/feed/new?lang=id"
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp dwFeedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, mod := range resp.Data.Items {
		for _, d := range mod.Items {
			dramas = append(dramas, models.Drama{
				BookID:       "dramawave:" + d.Key,
				Judul:        d.Title,
				Cover:        p.proxyImage(d.Cover),
				Deskripsi:    d.Desc,
				TotalEpisode: strconv.Itoa(d.EpCount),
			})
		}
	}
	return dramas, nil
}

func (p *DramaWaveProvider) Search(query string) ([]models.Drama, error) {
	// /search?q={q}&lang=id&page=1
	url := fmt.Sprintf("%s/search?q=%s&lang=id&page=1", DramaWaveAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	// Assuming search returns similar structure (Data.Items) or direct list?
	// Let's assume generic parsing or look at user request list (Get search/hot implies similar struct)
	// We'll try same dwFeedResponse first.
	var resp dwFeedResponse
	err = json.Unmarshal(body, &resp)
	if err != nil || len(resp.Data.Items) == 0 {
		// Fallback: maybe raw items list?
		var rawList struct {
			Data []dwDrama `json:"data"`
		}
		if json.Unmarshal(body, &rawList) == nil {
			var dramas []models.Drama
			for _, d := range rawList.Data {
				dramas = append(dramas, models.Drama{
					BookID:       "dramawave:" + d.Key,
					Judul:        d.Title,
					Cover:        p.proxyImage(d.Cover),
					Deskripsi:    d.Desc,
					TotalEpisode: strconv.Itoa(d.EpCount),
				})
			}
			return dramas, nil
		}
		return nil, err
	}

	var dramas []models.Drama
	for _, mod := range resp.Data.Items {
		for _, d := range mod.Items {
			dramas = append(dramas, models.Drama{
				BookID:       "dramawave:" + d.Key,
				Judul:        d.Title,
				Cover:        p.proxyImage(d.Cover),
				Deskripsi:    d.Desc,
				TotalEpisode: strconv.Itoa(d.EpCount),
			})
		}
	}
	return dramas, nil
}

func (p *DramaWaveProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// /dramas/{id}?lang=id
	url := fmt.Sprintf("%s/dramas/%s?lang=id", DramaWaveAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	var resp dwDetailResp
	// Try unmarshal
	if err := json.Unmarshal(body, &resp); err != nil {
		// Map fallback
		return nil, nil, err
	}
	d := resp.Data

	// If ID is empty in struct, maybe field name mismatch (e.g. "key" vs "id")
	// We use passed ID as backup
	dramaID := d.ID
	if dramaID == "" {
		dramaID = id
	}

	desc := d.Introduction
	if desc == "" {
		// Try parsing map for "brief" or "desc" if "introduction" is empty
		var raw map[string]interface{}
		json.Unmarshal(body, &raw)
		if data, ok := raw["data"].(map[string]interface{}); ok {
			desc, _ = data["desc"].(string)
		}
	}

	drama := models.Drama{
		BookID:       "dramawave:" + dramaID,
		Judul:        d.Title,
		Cover:        p.proxyImage(d.Cover),
		Deskripsi:    desc,
		TotalEpisode: strconv.Itoa(d.EpisodeCount),
	}

	var episodes []models.Episode
	// Check if Episodes populated. If not, generate range.
	if len(d.Episodes) == 0 && d.EpisodeCount > 0 {
		for i := 0; i < d.EpisodeCount; i++ {
			epNum := i + 1
			episodes = append(episodes, models.Episode{
				BookID:       "dramawave:" + dramaID,
				EpisodeIndex: i, // 0-based for internal
				EpisodeLabel: fmt.Sprintf("Episode %d", epNum),
			})
		}
	} else {
		for _, ep := range d.Episodes {
			// ep.Index usually 1-based in API
			episodes = append(episodes, models.Episode{
				BookID:       "dramawave:" + dramaID,
				EpisodeIndex: ep.Index - 1,
				EpisodeLabel: fmt.Sprintf("Episode %d", ep.Index),
			})
		}
	}

	return &drama, episodes, nil
}

func (p *DramaWaveProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// /dramas/{id}/play/{ep}?lang=id
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1 // 1-based API

	url := fmt.Sprintf("%s/dramas/%s/play/%d?lang=id", DramaWaveAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp dwStreamResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	videoURL := resp.Data.PlayUrl
	if videoURL == "" {
		videoURL = resp.Data.Url
	}
	if videoURL == "" {
		videoURL = resp.Data.M3u8
	}

	if videoURL == "" {
		// Inspect map
		var raw map[string]interface{}
		json.Unmarshal(body, &raw)
		if data, ok := raw["data"].(map[string]interface{}); ok {
			// Maybe it returns 'resource_url' ?
			if val, ok := data["resource_url"].(string); ok {
				videoURL = val
			}
		}
	}

	if videoURL == "" {
		return nil, fmt.Errorf("no video url found")
	}

	return &models.StreamData{
		BookID: "dramawave:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: videoURL,
			},
		},
	}, nil
}
