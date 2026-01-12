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

type DramaWaveProvider struct{}

const DramaWaveAPI = "https://dramabos.asia/api/dramawave/api/v1"

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

	// Comprehensive headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://dramabos.asia/")
	req.Header.Set("Origin", "https://dramabos.asia")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

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

// --- Models ---
type dwResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type dwList struct {
	Items []dwDrama `json:"items"` // Check if wrapped in items or direct array
	// If direct array, we handle it in Unmarshal
}

type dwDrama struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"` // or name
	Cover    string   `json:"cover"`
	Brief    string   `json:"brief"` // or summary/introduction
	Episodes int      `json:"episodes"`
	Tags     []string `json:"tags"`
}

type dwDetail struct {
	ID           int         `json:"id"`
	Title        string      `json:"title"`
	Cover        string      `json:"cover"`
	Introduction string      `json:"introduction"`
	EpisodeCount int         `json:"episode_count"` // or episodes
	EpisodeList  []dwEpisode `json:"episode_list"`  // Check key
}

type dwEpisode struct {
	Index int `json:"index"` // 0 or 1 based?
	// ... resource url if present
}

type dwStream struct {
	Url string `json:"url"` // video url
}

// Since structure is unknown, we map broadly or use robust parsing
// Based on ShortMax experience (`dramabos.asia/api/shortmax`), it uses:
// Data: { id, name, cover, summary, ... }

type dwGenericResponse struct {
	Data interface{} `json:"data"`
}

// --- Implementation ---

func (p *DramaWaveProvider) GetTrending() ([]models.Drama, error) {
	// /feed/popular?lang=id
	body, err := p.fetch(DramaWaveAPI + "/feed/popular?lang=id")
	if err != nil {
		return nil, err
	}

	// API returns {code, message, data: [...]}
	var raw struct {
		Code    int       `json:"code"`
		Message string    `json:"message"`
		Data    []dwDrama `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Data {
		dramas = append(dramas, models.Drama{
			BookID:       "dramawave:" + strconv.Itoa(d.ID),
			Judul:        d.Title,
			Cover:        p.proxyImage(d.Cover),
			Deskripsi:    d.Brief,
			TotalEpisode: strconv.Itoa(d.Episodes),
			Genre:        "",
		})
	}
	return dramas, nil
}

func (p *DramaWaveProvider) GetLatest(page int) ([]models.Drama, error) {
	// /feed/new?lang=id
	body, err := p.fetch(DramaWaveAPI + "/feed/new?lang=id")
	if err != nil {
		return nil, err
	}

	var raw struct {
		Code    int       `json:"code"`
		Message string    `json:"message"`
		Data    []dwDrama `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Data {
		dramas = append(dramas, models.Drama{
			BookID:       "dramawave:" + strconv.Itoa(d.ID),
			Judul:        d.Title,
			Cover:        p.proxyImage(d.Cover),
			Deskripsi:    d.Brief,
			TotalEpisode: strconv.Itoa(d.Episodes),
		})
	}
	return dramas, nil
}

func (p *DramaWaveProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/search?q=%s&lang=id&page=1", DramaWaveAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Code    int       `json:"code"`
		Message string    `json:"message"`
		Data    []dwDrama `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Data {
		dramas = append(dramas, models.Drama{
			BookID:       "dramawave:" + strconv.Itoa(d.ID),
			Judul:        d.Title,
			Cover:        p.proxyImage(d.Cover),
			Deskripsi:    d.Brief,
			TotalEpisode: strconv.Itoa(d.Episodes),
		})
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

	// Assuming detail has episodes inside? Or separate endpoint?
	// User snippet: /dramas/xuyr3DtXPt?lang=id
	// We'll assume it returns Detail object
	var raw struct {
		Data struct {
			ID           int    `json:"id"`
			Title        string `json:"title"`
			Cover        string `json:"cover"`
			Introduction string `json:"introduction"` // or brief
			EpisodeCount int    `json:"episode_count"`
			Episodes     []struct {
				Index int `json:"index"`
				// Assuming no direct video url here, usually
			} `json:"episodes"`
		} `json:"data"`
	}

	// Unmarshal might fail if structure is different.
	// But let's try.
	json.Unmarshal(body, &raw)
	// Retry with string ID if int fails?
	// User ID example "xuyr3DtXPt" is STRING.
	// So ID struct field must be string!

	// Redeclare struct just for Detail locally to be safe
	var rawStringID struct {
		Data struct {
			ID           string `json:"id"`
			Title        string `json:"title"`
			Cover        string `json:"cover"`
			Introduction string `json:"introduction"`
			EpisodeCount int    `json:"episode_count"` // or 'total'
			Episodes     []struct {
				Index int `json:"index"`
			} `json:"episodes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &rawStringID); err != nil {
		return nil, nil, err
	}

	d := rawStringID.Data
	drama := models.Drama{
		BookID:       "dramawave:" + d.ID,
		Judul:        d.Title,
		Cover:        p.proxyImage(d.Cover),
		Deskripsi:    d.Introduction,
		TotalEpisode: strconv.Itoa(d.EpisodeCount),
	}

	var episodes []models.Episode
	// If episodes list is empty, we generate default range based on count
	if len(d.Episodes) == 0 && d.EpisodeCount > 0 {
		for i := 0; i < d.EpisodeCount; i++ {
			episodes = append(episodes, models.Episode{
				BookID:       "dramawave:" + d.ID,
				EpisodeIndex: i,
				EpisodeLabel: fmt.Sprintf("Episode %d", i+1),
			})
		}
	} else {
		for _, ep := range d.Episodes {
			episodes = append(episodes, models.Episode{
				BookID:       "dramawave:" + d.ID,
				EpisodeIndex: ep.Index - 1,
				EpisodeLabel: fmt.Sprintf("Episode %d", ep.Index),
			})
		}
	}

	return &drama, episodes, nil
}

func (p *DramaWaveProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// Endpoint guess: /dramas/{id}/episodes/{epNo}?lang=id
	idx, _ := strconv.Atoi(epIndex)
	epNo := idx + 1 // 1-based

	url := fmt.Sprintf("%s/dramas/%s/episodes/%d?lang=id", DramaWaveAPI, id, epNo)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Data struct {
			URL string `json:"url"` // video url
			// Or maybe 'play_url', 'stream_url'
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	if raw.Data.URL == "" {
		// Try parsing generic map to find url key if 'url' failed?
		// For now simple return error
		return nil, fmt.Errorf("stream url empty or structure mismatch")
	}

	return &models.StreamData{
		BookID: "dramawave:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.Data.URL,
			},
		},
	}, nil
}
