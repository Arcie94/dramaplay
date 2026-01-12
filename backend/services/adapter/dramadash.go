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

// Home/Tabs response
type ddResponse struct {
	Data ddDataWrapper `json:"data"` // Or direct? Assume Wrapper
}
type ddDataWrapper struct {
	Items []ddItem `json:"items"` // Guess
}

// Actually user url: /api/home (Maybe list of categories?)
// /api/tabs/1 (Maybe a category?)

// Let's assume generic structure for simpler parsing
type ddGenericList struct {
	Data []ddItem `json:"data"`
}

type ddItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Cover string `json:"cover"`
	Desc  string `json:"summary"`
}

// Detail: /api/drama/44
type ddDetailResp struct {
	Data ddDetailData `json:"data"`
}
type ddDetailData struct {
	ID       int         `json:"id"`
	Title    string      `json:"title"`
	Cover    string      `json:"cover"`
	Summary  string      `json:"summary"`
	Episodes []ddEpisode `json:"episodes"`
}
type ddEpisode struct {
	ID    int `json:"id"`
	Index int `json:"index"` // or episode_number
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
	// Endpoint: /api/home (Likely returns categories with lists)
	// Or /api/tabs/1 (Likely 'Recommend' or similar)
	url := DramaDashAPI + "/tabs/1"
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp ddGenericList
	if err := json.Unmarshal(body, &resp); err != nil {
		// Fallback try raw list if no data wrapper
		var rawList []ddItem
		if json.Unmarshal(body, &rawList) == nil {
			resp.Data = rawList
		} else {
			return nil, err
		}
	}

	var dramas []models.Drama
	for _, b := range resp.Data {
		dramas = append(dramas, models.Drama{
			BookID:    "dramadash:" + strconv.Itoa(b.ID),
			Judul:     b.Title,
			Cover:     p.proxyImage(b.Cover),
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
	url := fmt.Sprintf("%s/search/%s", DramaDashAPI, query) // No query param? url path param?
	// User example: /api/search/cinta . Path param!

	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp ddGenericList
	if err := json.Unmarshal(body, &resp); err != nil {
		var rawList []ddItem
		if json.Unmarshal(body, &rawList) == nil {
			resp.Data = rawList
		} else {
			return nil, err
		}
	}

	var dramas []models.Drama
	for _, b := range resp.Data {
		dramas = append(dramas, models.Drama{
			BookID:    "dramadash:" + strconv.Itoa(b.ID),
			Judul:     b.Title,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Desc,
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
