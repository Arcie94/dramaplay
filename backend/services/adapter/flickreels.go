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
	// FlickReels URLs might already be absolute or relative?
	// Debug showed: https://zshipubcdn.farsunpteltd.com/playlet/1758184949_aS3nfwTBky.jpg
	// Perfect for wsrv.nl
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Models ---

// Home response: {"data":[{"list":[{...}]}]}
type frResponse struct {
	Data []frDataItem `json:"data"`
}
type frDataItem struct {
	List []frBook `json:"list"`
}

type frBook struct {
	ID        int    `json:"playlet_id"`
	Title     string `json:"title"`
	Cover     string `json:"cover"`
	Desc      string `json:"introduce"`
	UploadNum string `json:"upload_num"` // "88" (string)
}

// Detail: /drama/{id}?lang=6
// Response structure assumed from previous attempts or guess.
// If Home was weird, Detail might be too. But logic below likely used standard parsing.
// Let's assume detail is {data: {...}}
type frDetailResponse struct {
	Data frDetailData `json:"data"`
}
type frDetailData struct {
	ID           int         `json:"playlet_id"` // Check if playlet_id
	Title        string      `json:"title"`
	Cover        string      `json:"cover"`
	Introduction string      `json:"introduce"`
	EpisodeList  []frEpisode `json:"chapter_list"` // "chapter_list" is more likely given "chapter_type" in home
	// OR "list"? Debug of home showed "chapter_type":0.
	// Let's rely on standard fallback if this fails, but "chapter_list" is common.
	// Actually user didn't show detail response. I will stick to generic unmarshal map or guess.
	// Update: I'll use flexible struct tags or checks.
}

// Note: "episode_list" was my previous guess.

type frEpisode struct {
	Index int    `json:"index"`
	ID    int    `json:"chapter_id"`
	Url   string `json:"url"` // Streaming URL?
}

// --- Implementation ---

func (p *FlickReelsProvider) GetTrending() ([]models.Drama, error) {
	url := fmt.Sprintf("%s/home?page=1&page_size=20&lang=6", FlickReelsAPI)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp frResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 || len(resp.Data[0].List) == 0 {
		return nil, fmt.Errorf("no data found")
	}

	var dramas []models.Drama
	for _, b := range resp.Data[0].List {
		dramas = append(dramas, models.Drama{
			BookID:       "flickreels:" + strconv.Itoa(b.ID),
			Judul:        b.Title,
			Cover:        p.proxyImage(b.Cover),
			Deskripsi:    b.Desc,
			TotalEpisode: b.UploadNum,
		})
	}
	return dramas, nil
}

func (p *FlickReelsProvider) GetLatest(page int) ([]models.Drama, error) {
	return p.GetTrending()
}

func (p *FlickReelsProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/search?keyword=%s&lang=6", FlickReelsAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp frResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	// Search might return direct list or same structure
	// Assuming same structure for saftey, or check parsing
	var books []frBook
	if len(resp.Data) > 0 {
		books = resp.Data[0].List
	}

	var dramas []models.Drama
	for _, b := range books {
		dramas = append(dramas, models.Drama{
			BookID:       "flickreels:" + strconv.Itoa(b.ID),
			Judul:        b.Title,
			Cover:        p.proxyImage(b.Cover),
			Deskripsi:    b.Desc,
			TotalEpisode: b.UploadNum,
		})
	}
	return dramas, nil
}

func (p *FlickReelsProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	url := fmt.Sprintf("%s/drama/%s?lang=6", FlickReelsAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	// Use map interface to inspect structure if strict struct fails
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	// Manual extraction from map for robustness
	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("invalid detail response")
	}

	title, _ := data["title"].(string)
	cover, _ := data["cover"].(string)
	intro, _ := data["introduce"].(string)

	// Episodes ?
	// Check for "chapter_list", "episode_list", "list"
	var epList []interface{}
	if val, ok := data["chapter_list"].([]interface{}); ok {
		epList = val
	} else if val, ok := data["episode_list"].([]interface{}); ok {
		epList = val
	} else if val, ok := data["list"].([]interface{}); ok {
		epList = val
	}

	drama := models.Drama{
		BookID:       "flickreels:" + id,
		Judul:        title,
		Cover:        p.proxyImage(cover),
		Deskripsi:    intro,
		TotalEpisode: strconv.Itoa(len(epList)),
	}

	var episodes []models.Episode
	for i, e := range epList {
		epMap, _ := e.(map[string]interface{})
		// Usually "chapter_id", "title"
		epNum := i + 1

		episodes = append(episodes, models.Episode{
			BookID:       "flickreels:" + id,
			EpisodeIndex: i,
			EpisodeLabel: fmt.Sprintf("Episode %d", epNum),
		})

		// If video url is in detail list, we can log it but won't store it in model yet
		if u, ok := epMap["url"].(string); ok && u != "" {
			// Good
		}
	}

	return &drama, episodes, nil
}

func (p *FlickReelsProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)

	url := fmt.Sprintf("%s/drama/%s?lang=6", FlickReelsAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no data")
	}

	var epList []interface{}
	if val, ok := data["chapter_list"].([]interface{}); ok {
		epList = val
	} else if val, ok := data["episode_list"].([]interface{}); ok {
		epList = val
	} else if val, ok := data["list"].([]interface{}); ok {
		epList = val
	}

	if len(epList) <= idx {
		return nil, fmt.Errorf("episode not found")
	}

	epMap, _ := epList[idx].(map[string]interface{})
	videoURL, _ := epMap["url"].(string)
	if videoURL == "" {
		videoURL, _ = epMap["video_url"].(string) // try alias
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
