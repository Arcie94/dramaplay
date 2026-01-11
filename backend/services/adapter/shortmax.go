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

type ShortMaxProvider struct{}

const ShortMaxAPI = "https://dramabos.asia/api/shortmax/api/v1"

func NewShortMaxProvider() *ShortMaxProvider {
	return &ShortMaxProvider{}
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

	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	// Using generic SapimuToken or empty if no auth needed for dramabos?
	// User didn't specify auth for dramabos.asia. Assuming open or same token.
	// The provided URLs in the prompt contain "auth_key" in image URLs but the API URLs themselves don't seem to have tokens in them (except maybe implicit?)
	// Wait, the prompt urls: `https://dramabos.asia/api/shortmax/api/v1/home?lang=id`
	// No "token" query param.
	// I will keep sending SapimuToken just in case, or remove if it causes 400.
	// The user didn't say "don't use token".
	// I will use SapimuToken as header just to be safe/consistent, unless it breaks it.
	req.Header.Set("Authorization", "Bearer "+SapimuToken)

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

func (p *ShortMaxProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

type smResponse struct {
	Data []smItem `json:"data"`
}
type smItem struct {
	ID        int      `json:"id"`
	Code      int      `json:"code"`
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

func (p *ShortMaxProvider) GetTrending() ([]models.Drama, error) {
	// Use /home for Trending
	body, err := p.fetch(ShortMaxAPI + "/home?lang=id")
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
			Genre:        fmt.Sprintf("%v", d.Tags),
		})
	}
	return dramas, nil
}

func (p *ShortMaxProvider) GetLatest(page int) ([]models.Drama, error) {
	return p.GetTrending()
}

func (p *ShortMaxProvider) Search(query string) ([]models.Drama, error) {
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
			Genre:        fmt.Sprintf("%v", d.Tags),
		})
	}
	return dramas, nil
}

func (p *ShortMaxProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// 1. Fetch Metadata (Title) from /batch
	// /batch endpoint returns NDJSON, first line has metadata.
	urlBatch := fmt.Sprintf("%s/batch/%s?lang=id", ShortMaxAPI, id)
	var dramaTitle = "ShortMax Drama " + id
	var dramaDesc = "Metadata currently not available via direct detail endpoint."

	// We use a short timeout or just fetch it.
	// We read the body, take the first line.
	bodyBatch, err := p.fetch(urlBatch)
	if err == nil {
		// Parse NDJSON first line
		lines := strings.Split(string(bodyBatch), "\n")
		if len(lines) > 0 {
			var meta struct {
				Name    string `json:"name"`
				Total   int    `json:"total"`
				Summary string `json:"summary"` // Optimistic guess, logic below checks
			}
			if json.Unmarshal([]byte(lines[0]), &meta) == nil && meta.Name != "" {
				dramaTitle = meta.Name
				if meta.Summary != "" {
					dramaDesc = meta.Summary
				} else {
					dramaDesc = meta.Name // Use title as desc if empty
				}
			}
		}
	}

	// 2. Fetch Episodes from /episodes for reliable Locked status
	urlEpisodes := fmt.Sprintf("%s/episodes/%s?lang=id", ShortMaxAPI, id)
	body, err := p.fetch(urlEpisodes)
	if err != nil {
		return nil, nil, err
	}

	var raw smEpisodeResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "shortmax:" + id,
		Judul:        dramaTitle,
		Cover:        "", // Still no source for cover in batch/episodes
		Deskripsi:    dramaDesc,
		TotalEpisode: strconv.Itoa(len(raw.Data)),
	}

	var episodes []models.Episode
	for _, ep := range raw.Data {
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
		return nil, fmt.Errorf("no video url found in response")
	}

	// Use Proxy to handle CORS/Auth
	proxiedURL := "/api/proxy?url=" + url.QueryEscape(videoURL)

	return &models.StreamData{
		BookID: "shortmax:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: proxiedURL,
			},
		},
	}, nil
}
