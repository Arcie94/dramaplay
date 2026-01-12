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
	"time"
)

type ShortMaxProvider struct {
	client *http.Client
}

const ShortMaxAPI = "https://dramabos.asia/api/shortmax/api/v1"

func NewShortMaxProvider() *ShortMaxProvider {
	return &ShortMaxProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://dramabos.asia/")
	req.Header.Set("Origin", "https://dramabos.asia")

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

func (p *ShortMaxProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Internal Models ---

type smResponse struct {
	Data []smItem `json:"data"`
}

type smItem struct {
	ID        int      `json:"id"`
	Code      int      `json:"code"` // Sometimes ID is in Code
	Name      string   `json:"name"`
	Cover     string   `json:"cover"`
	Episodes  int      `json:"episodes"`
	Summary   string   `json:"summary"`
	Tags      []string `json:"tags"`
	Favorites int      `json:"favorites"`
}

type smEpisodeResponse struct {
	Data []smEpisode `json:"data"`
}

type smEpisode struct {
	ID      int  `json:"id"`
	Episode int  `json:"episode"`
	Locked  bool `json:"locked"`
}

type smPlayResponse struct {
	Data struct {
		Video struct {
			Video1080 string `json:"video_1080"`
			Video720  string `json:"video_720"`
			Video480  string `json:"video_480"`
		} `json:"video"`
	} `json:"data"`
}

type smBatchMeta struct {
	// Trying to cover common list/item fields if batch returns a list or item
	// Usually batch is [ {project_info}, {episode_list...} ] or similar line-delimited
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Cover   string `json:"cover"`
}

// --- Implementation ---

func (p *ShortMaxProvider) GetTrending() ([]models.Drama, error) {
	// Use /home?lang=id
	body, err := p.fetch(ShortMaxAPI + "/home?lang=id")
	if err != nil {
		return nil, err
	}

	// Try parsing as standard wrapper
	var raw smResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Data {
		dramas = append(dramas, models.Drama{
			BookID:       "shortmax:" + strconv.Itoa(d.ID),
			Judul:        d.Name,
			Cover:        p.proxyImage(d.Cover),
			Deskripsi:    d.Summary,
			TotalEpisode: strconv.Itoa(d.Episodes),
			Likes:        strconv.Itoa(d.Favorites),
			Genre:        strings.Join(d.Tags, ", "),
		})
	}
	return dramas, nil
}

func (p *ShortMaxProvider) GetLatest(page int) ([]models.Drama, error) {
	// ShortMax doesn't seem to have specific pagination in provided endpoints
	// Fallback to Trending/Home
	return p.GetTrending()
}

func (p *ShortMaxProvider) Search(query string) ([]models.Drama, error) {
	// Endpoint: /search?q={q}&lang=id&page=1
	urlSearch := fmt.Sprintf("%s/search?q=%s&lang=id&page=1", ShortMaxAPI, url.QueryEscape(query))
	body, err := p.fetch(urlSearch)
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
			BookID:       "shortmax:" + strconv.Itoa(d.ID),
			Judul:        d.Name,
			Cover:        p.proxyImage(d.Cover),
			Deskripsi:    d.Summary,
			TotalEpisode: strconv.Itoa(d.Episodes),
			Likes:        strconv.Itoa(d.Favorites),
			Genre:        strings.Join(d.Tags, ", "),
		})
	}
	return dramas, nil
}

func (p *ShortMaxProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// Strategy:
	// 1. Fetch metadata from /batch/{id}?lang=id
	//    Usually this contains details.
	//    If batch fails or is empty, we might only have episodes from /episodes.

	urlBatch := fmt.Sprintf("%s/batch/%s?lang=id", ShortMaxAPI, id)

	dramaTitle := "ShortMax Drama " + id
	dramaDesc := "No description available"
	dramaCover := ""

	// Fetch batch (optimistic)
	if bodyBatch, err := p.fetch(urlBatch); err == nil {
		// Try to handle both NDJSON or standard JSON array
		sBody := string(bodyBatch)

		// If response starts with [ or {, good.
		// If NDJSON, split newline.

		var firstItem smItem // Re-use smItem as it matches drama fields
		var successParse bool

		// Attempt 1: Parse as single object
		if json.Unmarshal(bodyBatch, &firstItem) == nil && firstItem.Name != "" {
			successParse = true
		} else {
			// Attempt 2: Parse as list, take first
			var list []smItem
			if json.Unmarshal(bodyBatch, &list) == nil && len(list) > 0 {
				firstItem = list[0]
				successParse = true
			}
		}

		if successParse {
			dramaTitle = firstItem.Name
			dramaDesc = firstItem.Summary
			dramaCover = firstItem.Cover
		} else {
			// NDJSON fallback (common in ShortMax APIs)
			lines := strings.Split(sBody, "\n")
			if len(lines) > 0 {
				// Try parsing first line
				if json.Unmarshal([]byte(lines[0]), &firstItem) == nil && firstItem.Name != "" {
					dramaTitle = firstItem.Name
					dramaDesc = firstItem.Summary
					dramaCover = firstItem.Cover
				}
			}
		}
	}

	// 2. Fetch Episodes List: /episodes/{id}?lang=id
	urlEp := fmt.Sprintf("%s/episodes/%s?lang=id", ShortMaxAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return nil, nil, err
	}

	var rawEp smEpisodeResponse
	if err := json.Unmarshal(bodyEp, &rawEp); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "shortmax:" + id,
		Judul:        dramaTitle,
		Cover:        p.proxyImage(dramaCover),
		Deskripsi:    dramaDesc,
		TotalEpisode: strconv.Itoa(len(rawEp.Data)),
	}

	var episodes []models.Episode
	for _, ep := range rawEp.Data {
		episodes = append(episodes, models.Episode{
			BookID:       "shortmax:" + id,
			EpisodeIndex: ep.Episode - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}

	return &drama, episodes, nil
}

func (p *ShortMaxProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1

	urlPlay := fmt.Sprintf("%s/play/%s?lang=id&ep=%d", ShortMaxAPI, id, epNum)
	body, err := p.fetch(urlPlay)
	if err != nil {
		return nil, err
	}

	var raw smPlayResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	// Pick best quality
	videoURL := raw.Data.Video.Video1080
	if videoURL == "" {
		videoURL = raw.Data.Video.Video720
	}
	if videoURL == "" {
		videoURL = raw.Data.Video.Video480
	}

	if videoURL == "" {
		return nil, fmt.Errorf("no video url found")
	}

	return &models.StreamData{
		BookID: "shortmax:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: videoURL,
			},
		},
	}, nil
}
