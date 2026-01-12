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

type FlickReelsProvider struct {
	client *http.Client
}

const FlickReelsAPI = "https://dramabos.asia/api/flick"

func NewFlickReelsProvider() *FlickReelsProvider {
	return &FlickReelsProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
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

func (p *FlickReelsProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Models ---
// Endpoint: /home?page=1&page_size=10&lang=6
// Response guess: { code: 0, msg: "success", data: [...] } or { data: [...] }
// Based on others: likely { data: { list: [...] } } or { data: [...] }

// Let's assume standard response based on query params pattern
type frResponse struct {
	Data frListWrapper `json:"data"`
}

// Or maybe Data is direct list?
// Let's create a flexible unmarshal logic or assume wrap first.
// Actually, user provided /home params `page=1`. So likely a list response.

type frListWrapper struct {
	List []frBook `json:"list"` // Guessing 'list' key
}

// If it's just a raw list in Data:
type frResponseList struct {
	Data []frBook `json:"data"`
}

type frBook struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Cover    string `json:"cover"`
	Desc     string `json:"introduction"` // or summary
	Episodes int    `json:"episodes"`
}

// Detail: /drama/{id}?lang=6
type frDetailResponse struct {
	Data frDetailData `json:"data"`
}
type frDetailData struct {
	ID           int         `json:"id"`
	Title        string      `json:"title"`
	Cover        string      `json:"cover"`
	Introduction string      `json:"introduction"`
	EpisodeList  []frEpisode `json:"episode_list"` // Guessing key
	// If streaming links are here?
}

type frEpisode struct {
	Index int    `json:"index"`
	Url   string `json:"url"` // Streaming URL?
}

// --- Implementation ---

func (p *FlickReelsProvider) GetTrending() ([]models.Drama, error) {
	// Endpoint: /home?page=1&page_size=10&lang=6
	// lang=6 (Indonesia?)
	url := fmt.Sprintf("%s/home?page=1&page_size=20&lang=6", FlickReelsAPI)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	// Try unmarshal
	var resp frResponseList
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Data) == 0 {
		return nil, fmt.Errorf("failed to parse home")
	}

	var dramas []models.Drama
	for _, b := range resp.Data {
		dramas = append(dramas, models.Drama{
			BookID:    "flickreels:" + strconv.Itoa(b.ID),
			Judul:     b.Title,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Desc,
		})
	}
	return dramas, nil
}

func (p *FlickReelsProvider) GetLatest(page int) ([]models.Drama, error) {
	// /latest?page=1&page_size=10&lang=6
	url := fmt.Sprintf("%s/latest?page=%d&page_size=20&lang=6", FlickReelsAPI, page)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp frResponseList
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range resp.Data {
		dramas = append(dramas, models.Drama{
			BookID:    "flickreels:" + strconv.Itoa(b.ID),
			Judul:     b.Title,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Desc,
		})
	}
	return dramas, nil
}

func (p *FlickReelsProvider) Search(query string) ([]models.Drama, error) {
	// /search?keyword=cinta&lang=6
	url := fmt.Sprintf("%s/search?keyword=%s&lang=6", FlickReelsAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp frResponseList
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range resp.Data {
		dramas = append(dramas, models.Drama{
			BookID:    "flickreels:" + strconv.Itoa(b.ID),
			Judul:     b.Title,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Desc,
		})
	}
	return dramas, nil
}

func (p *FlickReelsProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// /drama/{id}?lang=6
	url := fmt.Sprintf("%s/drama/%s?lang=6", FlickReelsAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	var resp frDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, err
	}
	d := resp.Data

	drama := models.Drama{
		BookID:       "flickreels:" + strconv.Itoa(d.ID),
		Judul:        d.Title,
		Cover:        p.proxyImage(d.Cover),
		Deskripsi:    d.Introduction,
		TotalEpisode: strconv.Itoa(len(d.EpisodeList)),
	}

	var episodes []models.Episode
	for i, ep := range d.EpisodeList {
		// Use index if ep.Index is 0
		idx := ep.Index
		if idx == 0 {
			idx = i + 1
		}

		episodes = append(episodes, models.Episode{
			BookID:       "flickreels:" + id,
			EpisodeIndex: idx - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", idx),
		})
	}

	return &drama, episodes, nil
}

func (p *FlickReelsProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// Need stream URL.
	// As user didn't provide /play endpoint, I assume stream URL is inside EpisodeList from GetDetail.
	// So I need to fetch GetDetail again or we need to cache result? Adapter doesn't cache.
	// So we fetch detail.

	idx, _ := strconv.Atoi(epIndex)

	url := fmt.Sprintf("%s/drama/%s?lang=6", FlickReelsAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp frDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	// Find episode
	var videoURL string
	targetEp := idx + 1 // 1-based usually

	// Try to find by index matching
	if len(resp.Data.EpisodeList) > idx {
		// Verify index
		ep := resp.Data.EpisodeList[idx]
		// If api returns "url" or "video_url"
		videoURL = ep.Url
	}

	if videoURL == "" {
		return nil, fmt.Errorf("video url not found in detail endpoint")
	}

	return &models.StreamData{
		BookID: "flickreels:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: videoURL,
			},
		},
	}, nil
}
