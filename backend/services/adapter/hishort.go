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

type HiShortProvider struct {
	client *http.Client
}

const HiShortAPI = "https://dramabos.asia/api/hishort/api/v1"

func NewHiShortProvider() *HiShortProvider {
	return &HiShortProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *HiShortProvider) GetID() string {
	return "hishort"
}

func (p *HiShortProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *HiShortProvider) fetch(targetURL string) ([]byte, error) {
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

func (p *HiShortProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Models ---
// Structure based on DEBUG: { pageNum: 1, source: [...] }
type hsResponse struct {
	Source []hsItem `json:"source"`
}

type hsItem struct {
	ID    int    `json:"vidId"`
	Title string `json:"vidName"`
	Cover string `json:"coverUrl"`
	Desc  string `json:"vidDescribe"`
}

// /video/{id} (Detail + Play?)
type hsDetailResp struct {
	Data hsDetailData `json:"data"` // Hopefully detail follows Data wrapper? If not, need debug.
	// But let's assume standard wrapper for detail because usually list and detail API formats differ slightly.
}
type hsDetailData struct {
	ID       int    `json:"vidId"`       // or id
	Title    string `json:"vidName"`     // or title
	Cover    string `json:"coverUrl"`    // or cover
	Desc     string `json:"vidDescribe"` // or desc
	VideoUrl string `json:"url"`         // Streaming url
}

// /video/{id}/playlist
type hsPlaylistResp struct {
	Data []hsEpisode `json:"data"`
}
type hsEpisode struct {
	ID int `json:"id"`
	No int `json:"no"` // Episode number
}

// --- Implementation ---

func (p *HiShortProvider) GetTrending() ([]models.Drama, error) {
	// /home?module=12&page=1
	url := HiShortAPI + "/home?module=12&page=1"
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp hsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range resp.Source {
		dramas = append(dramas, models.Drama{
			BookID:    "hishort:" + strconv.Itoa(b.ID),
			Judul:     b.Title,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Desc,
		})
	}
	return dramas, nil
}

func (p *HiShortProvider) GetLatest(page int) ([]models.Drama, error) {
	return p.GetTrending()
}

func (p *HiShortProvider) Search(query string) ([]models.Drama, error) {
	// /search?q=love
	url := fmt.Sprintf("%s/search?q=%s", HiShortAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	// Assuming search returns same "source" [] list
	var resp hsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range resp.Source {
		dramas = append(dramas, models.Drama{
			BookID:    "hishort:" + strconv.Itoa(b.ID),
			Judul:     b.Title,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Desc,
		})
	}
	return dramas, nil
}

func (p *HiShortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// /video/{id} for detail
	url := fmt.Sprintf("%s/video/%s", HiShortAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	// Parsing detail using generic map to find keys
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	// Locate Data object
	dataMap := raw
	if d, ok := raw["data"].(map[string]interface{}); ok {
		dataMap = d
	}

	// Try keys: vidId, id, etc.
	title, _ := dataMap["vidName"].(string)
	if title == "" {
		title, _ = dataMap["title"].(string)
	}

	cover, _ := dataMap["coverUrl"].(string)
	if cover == "" {
		cover, _ = dataMap["cover"].(string)
	}

	desc, _ := dataMap["vidDescribe"].(string)
	if desc == "" {
		desc, _ = dataMap["introduction"].(string)
	}

	drama := models.Drama{
		BookID:    "hishort:" + id,
		Judul:     title,
		Cover:     p.proxyImage(cover),
		Deskripsi: desc,
	}

	// Playlist: /video/{id}/playlist
	urlPlay := fmt.Sprintf("%s/video/%s/playlist", HiShortAPI, id)
	bodyPlay, err := p.fetch(urlPlay)
	if err == nil {
		// Try parsing "data" keys
		var respPlay struct {
			Data []map[string]interface{} `json:"data"`
		}
		if json.Unmarshal(bodyPlay, &respPlay) == nil {
			drama.TotalEpisode = strconv.Itoa(len(respPlay.Data))
			var episodes []models.Episode
			for i := range respPlay.Data {
				epNum := i + 1
				episodes = append(episodes, models.Episode{
					BookID:       "hishort:" + id,
					EpisodeIndex: i,
					EpisodeLabel: fmt.Sprintf("Episode %d", epNum),
				})
			}
			return &drama, episodes, nil
		}
	}

	return &drama, nil, nil
}

func (p *HiShortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// /video/{id}?ep={ep}
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1

	url := fmt.Sprintf("%s/video/%s?ep=%d", HiShortAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	dataMap := raw
	if d, ok := raw["data"].(map[string]interface{}); ok {
		dataMap = d
	}

	videoURL, _ := dataMap["url"].(string)
	if videoURL == "" {
		videoURL, _ = dataMap["videoUrl"].(string)
	}

	if videoURL == "" {
		return nil, fmt.Errorf("no video url")
	}

	return &models.StreamData{
		BookID: "hishort:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: videoURL,
			},
		},
	}, nil
}
