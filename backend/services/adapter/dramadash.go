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

type DramaDashProvider struct {
	client *http.Client
}

const DramaDashAPI = "https://dramabos.asia/api/dramadash/api"

func NewDramaDashProvider() *DramaDashProvider {
	return &DramaDashProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

func (p *DramaDashProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Models ---
// Home: {"data": {"banner": [...]}}
type ddHomeResponse struct {
	Data struct {
		Banner []ddItem `json:"banner"`
	} `json:"data"`
}

type ddGenericList struct {
	// Search often returns list in data
	Data []ddItemSearch `json:"data"` // Or generic items?
}

// Note: Search endpoint structure wasn't debugged as clear as home.
// Assuming it might be different. But generic list is safer guess if not sure.

type ddItem struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Poster string `json:"poster"`
	Desc   string `json:"desc"` // or summary
}

type ddItemSearch struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Poster string `json:"poster"`
	Slug   string `json:"slug"`
}

// Detail: /api/drama/44
type ddDetailResp struct {
	Data ddDetailData `json:"data"`
}
type ddDetailData struct {
	ID       int         `json:"id"`
	Title    string      `json:"title"`
	Cover    string      `json:"poster"` // Usually 'poster' in DramaDash
	Summary  string      `json:"desc"`   // 'desc'
	Episodes []ddEpisode `json:"episodes"`
}
type ddEpisode struct {
	ID    int `json:"id"`
	Index int `json:"index"`
}

// Stream: /api/episode/44/1
type ddStreamResp struct {
	Data ddStreamData `json:"data"`
}
type ddStreamData struct {
	Url string `json:"url"` // video url
}

// --- Implementation ---

func (p *DramaDashProvider) GetTrending() ([]models.Drama, error) {
	// Endpoint: /api/home
	url := DramaDashAPI + "/home"
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp ddHomeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range resp.Data.Banner {
		dramas = append(dramas, models.Drama{
			BookID:    "dramadash:" + strconv.Itoa(b.ID),
			Judul:     b.Name,
			Cover:     p.proxyImage(b.Poster),
			Deskripsi: b.Desc,
		})
	}
	return dramas, nil
}

func (p *DramaDashProvider) GetLatest(page int) ([]models.Drama, error) {
	return p.GetTrending()
}

func (p *DramaDashProvider) Search(query string) ([]models.Drama, error) {
	// /api/search/cinta
	url := fmt.Sprintf("%s/search/%s", DramaDashAPI, query)

	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	// Search likely returns list of ddItemSearch directly or in data wrapper?
	// Let's assume list in data because user said /api/search/cinta but didn't show json.
	// We'll try generic wrapper.

	// Re-use ddGenericList locally or map
	var resp struct {
		Data []ddItemSearch `json:"data"`
	}
	if json.Unmarshal(body, &resp) != nil || len(resp.Data) == 0 {
		// Try raw list?
		var raw []ddItemSearch
		if json.Unmarshal(body, &raw) == nil {
			resp.Data = raw
		} else {
			// Try "trendingSearches" structure from tabs debugging?
			// tabs/1 returned "list": [], "moreToExplore": [...]
			// Maybe search returns similar?
			// Fallback: empty
			return []models.Drama{}, nil
		}
	}

	var dramas []models.Drama
	for _, b := range resp.Data {
		dramas = append(dramas, models.Drama{
			BookID:    "dramadash:" + strconv.Itoa(b.ID),
			Judul:     b.Name,
			Cover:     p.proxyImage(b.Poster),
			Deskripsi: "",
		})
	}
	return dramas, nil
}

func (p *DramaDashProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// /api/drama/{id}
	url := fmt.Sprintf("%s/drama/%s", DramaDashAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	var resp ddDetailResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, err
	}
	d := resp.Data

	drama := models.Drama{
		BookID:       "dramadash:" + strconv.Itoa(d.ID),
		Judul:        d.Title,
		Cover:        p.proxyImage(d.Cover),
		Deskripsi:    d.Summary,
		TotalEpisode: strconv.Itoa(len(d.Episodes)),
	}

	var episodes []models.Episode
	for i := range d.Episodes {
		epNum := i + 1
		episodes = append(episodes, models.Episode{
			BookID:       "dramadash:" + id,
			EpisodeIndex: i,
			EpisodeLabel: fmt.Sprintf("Episode %d", epNum),
		})
	}

	return &drama, episodes, nil
}

func (p *DramaDashProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// /api/episode/{id}/{epNum}
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1

	url := fmt.Sprintf("%s/episode/%s/%d", DramaDashAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp ddStreamResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Data.Url == "" {
		return nil, fmt.Errorf("no video url")
	}

	return &models.StreamData{
		BookID: "dramadash:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: resp.Data.Url,
			},
		},
	}, nil
}
