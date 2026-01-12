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
// /home?module=12&page=1
type hsResponse struct {
	Data []hsItem `json:"data"` // Assuming list
}
type hsItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"` // or name
	Cover string `json:"cover"`
	Desc  string `json:"introduction"` // or summary
}

// /video/{id} (Detail + Play?)
type hsDetailResp struct {
	Data hsDetailData `json:"data"`
}
type hsDetailData struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Cover    string `json:"cover"`
	Desc     string `json:"introduction"`
	VideoUrl string `json:"url"` // Streaming url usually here if it's /video endpoint
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
	for _, b := range resp.Data {
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

	var resp hsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range resp.Data {
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

	var resp hsDetailResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, err
	}
	d := resp.Data

	drama := models.Drama{
		BookID:    "hishort:" + strconv.Itoa(d.ID),
		Judul:     d.Title,
		Cover:     p.proxyImage(d.Cover),
		Deskripsi: d.Desc,
	}

	// Playlist: /video/{id}/playlist
	urlPlay := fmt.Sprintf("%s/video/%s/playlist", HiShortAPI, id)
	bodyPlay, err := p.fetch(urlPlay)
	if err == nil {
		var respPlay hsPlaylistResp
		if json.Unmarshal(bodyPlay, &respPlay) == nil {
			drama.TotalEpisode = strconv.Itoa(len(respPlay.Data))
			var episodes []models.Episode
			for i, _ := range respPlay.Data {
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

	var resp hsDetailResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Data.VideoUrl == "" {
		return nil, fmt.Errorf("no video url")
	}

	return &models.StreamData{
		BookID: "hishort:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: resp.Data.VideoUrl,
			},
		},
	}, nil
}
