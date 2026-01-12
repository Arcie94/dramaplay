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

type MeloloProvider struct{}

const MeloloAPI = "https://dramabos.asia/api/melolo/api/v1"

func NewMeloloProvider() *MeloloProvider {
	return &MeloloProvider{}
}

func (p *MeloloProvider) GetID() string {
	return "melolo"
}

func (p *MeloloProvider) IsCompatibleID(id string) bool {
	return true // Relies on Manager routing
}

func (p *MeloloProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	// Default headers to mimic a browser/app
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	// No auth needed for dramabos.asia

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

func (p *MeloloProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	// Proxy image through wsrv.nl for optimization and CORS
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Internal Models ---

// Home/Trending response
type mHomeResponse struct {
	Code    int      `json:"code"`
	Offset  int      `json:"offset"`
	Count   int      `json:"count"`
	HasMore bool     `json:"has_more"`
	Data    []mDrama `json:"data"`
}

type mDrama struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Cover    string `json:"cover"`
	Author   string `json:"author"`
	Episodes string `json:"episodes"`
	Intro    string `json:"intro"`
}

// Search response
type mSearchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		SearchData []mDrama `json:"search_data"`
		HasMore    bool     `json:"has_more"`
	} `json:"data"`
}

// Detail response
type mDetailResponse struct {
	Code     int    `json:"code"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Episodes int    `json:"episodes"`
	Cover    string `json:"cover"`
	Intro    string `json:"intro"`
	Videos   []struct {
		Vid      string `json:"vid"`
		Episode  int    `json:"episode"`
		Duration int    `json:"duration"`
	} `json:"videos"`
}

// Stream/Video response
type mStreamResponse struct {
	URL    string `json:"url"`
	Backup string `json:"backup"`
	List   []struct {
		Definition string `json:"definition"`
		URL        string `json:"url"`
	} `json:"list"`
}

// --- Implementation ---

func (p *MeloloProvider) GetTrending() ([]models.Drama, error) {
	// Use /home for Trending
	body, err := p.fetch(MeloloAPI + "/home?offset=0&count=20")
	if err != nil {
		return nil, err
	}

	var raw mHomeResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Data {
		dramas = append(dramas, models.Drama{
			BookID:       "melolo:" + d.ID,
			Judul:        d.Name,
			Cover:        p.proxyImage(d.Cover),
			Deskripsi:    d.Intro,
			TotalEpisode: d.Episodes,
			Genre:        "",
		})
	}
	return dramas, nil
}

func (p *MeloloProvider) GetLatest(page int) ([]models.Drama, error) {
	// Reuse home for latest as well
	return p.GetTrending()
}

func (p *MeloloProvider) Search(query string) ([]models.Drama, error) {
	urlSearch := fmt.Sprintf("%s/search?q=%s&offset=0&count=20", MeloloAPI, url.QueryEscape(query))
	body, err := p.fetch(urlSearch)
	if err != nil {
		return nil, err
	}

	var raw mSearchResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		// Return empty list on parse error
		return []models.Drama{}, nil
	}

	var dramas []models.Drama
	for _, d := range raw.Data.SearchData {
		dramas = append(dramas, models.Drama{
			BookID:       "melolo:" + d.ID,
			Judul:        d.Name,
			Cover:        p.proxyImage(d.Cover),
			Deskripsi:    d.Intro,
			TotalEpisode: d.Episodes,
			Genre:        "",
		})
	}
	return dramas, nil
}

func (p *MeloloProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// Use /detail/{id}
	urlDetail := fmt.Sprintf("%s/detail/%s", MeloloAPI, id)
	body, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}

	var raw mDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	// Parse Drama Info (flat structure now)
	drama := models.Drama{
		BookID:       "melolo:" + raw.ID,
		Judul:        raw.Title,
		Cover:        p.proxyImage(raw.Cover),
		Deskripsi:    raw.Intro,
		TotalEpisode: strconv.Itoa(raw.Episodes),
	}

	// Parse Episodes from videos array
	var episodes []models.Episode
	for _, vid := range raw.Videos {
		episodes = append(episodes, models.Episode{
			BookID:       "melolo:" + raw.ID,
			EpisodeIndex: vid.Episode - 1, // API is 1-based, convert to 0-based
			EpisodeLabel: fmt.Sprintf("Episode %d", vid.Episode),
		})
	}

	return &drama, episodes, nil
}

func (p *MeloloProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// 1. Fetch Detail to map Index -> VID
	urlDetail := fmt.Sprintf("%s/detail/%s", MeloloAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, err
	}

	var rawDetail mDetailResponse
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, err
	}

	idx, _ := strconv.Atoi(epIndex) // 0-based from frontend
	targetEpisode := idx + 1        // 1-based API episode number

	var targetVid string
	for _, vid := range rawDetail.Videos {
		if vid.Episode == targetEpisode {
			targetVid = vid.Vid
			break
		}
	}

	if targetVid == "" {
		return nil, fmt.Errorf("episode index %d not found", idx)
	}

	// 2. Fetch Video Stream using /video/{vid}
	urlStream := fmt.Sprintf("%s/video/%s", MeloloAPI, targetVid)
	bodyStream, err := p.fetch(urlStream)
	if err != nil {
		return nil, err
	}

	var rawStream mStreamResponse
	if err := json.Unmarshal(bodyStream, &rawStream); err != nil {
		return nil, err
	}

	// Use the main URL field
	videoURL := rawStream.URL
	if videoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}

	// Force HTTPS
	finalUrl := strings.Replace(videoURL, "http://", "https://", 1)

	return &models.StreamData{
		BookID: "melolo:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: finalUrl,
			},
		},
	}, nil
}
