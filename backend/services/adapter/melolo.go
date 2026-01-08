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

const MeloloAPI = "https://sapimu.au/melolo"

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
	// Auth Token
	req.Header.Set("x-token", "0ebd6cfdd8054d2a90aa2851532645211aeaf189fa1aed62c53e5fd735af8649")

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

type mBookmallResponse struct {
	Cell struct {
		Books []mBook `json:"books"`
	} `json:"cell"`
}

type mBook struct {
	BookID      string   `json:"book_id"`
	BookName    string   `json:"book_name"` // For Bookmall
	Title       string   `json:"title"`     // For Search
	ThumbURL    string   `json:"thumb_url"` // For Bookmall
	Cover       string   `json:"cover"`     // For Search & Detail
	Abstract    string   `json:"abstract"`
	SerialCount string   `json:"serial_count"`
	StatInfos   []string `json:"stat_infos"` // Genre list
}

type mSearchResponse struct {
	Items []mBook `json:"items"`
}

type mDetailResponse struct {
	Series struct {
		SeriesID     int64  `json:"series_id"`
		Title        string `json:"title"`
		Intro        string `json:"intro"`
		Cover        string `json:"cover"`
		EpisodeCount int    `json:"episode_count"`
	} `json:"series"`
	Episodes []mEpisodeItem `json:"episodes"`
}

type mEpisodeItem struct {
	Index int64  `json:"index"`
	Vid   string `json:"vid"`
	Cover string `json:"cover"`
}

type mStreamResponse struct {
	MainURL string `json:"main_url"`
	// Sometimes it might be directly in root or in 'data' depending on endpoint variation,
	// but user example shows flat JSON with "main_url" for Melolo Video URL response.
}

// --- Implementation ---

func (p *MeloloProvider) GetTrending() ([]models.Drama, error) {
	// Use /bookmall for trending/home content
	body, err := p.fetch(MeloloAPI + "/bookmall")
	if err != nil {
		return nil, err
	}

	var raw mBookmallResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range raw.Cell.Books {
		// Bookmall uses BookName and ThumbURL
		dramas = append(dramas, models.Drama{
			BookID:       "melolo:" + b.BookID,
			Judul:        b.BookName,
			Cover:        p.proxyImage(b.ThumbURL),
			Deskripsi:    b.Abstract,
			TotalEpisode: b.SerialCount,
			Genre:        strings.Join(b.StatInfos, ", "),
		})
	}
	return dramas, nil
}

func (p *MeloloProvider) GetLatest(page int) ([]models.Drama, error) {
	// Reuse Bookmall for latest as well, maybe just return same list or minimal shuffle if needed.
	// API doesn't seem to have explicit 'latest' param in user snippet, defaulting to bookmall.
	return p.GetTrending()
}

func (p *MeloloProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/search?query=%s", MeloloAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw mSearchResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		// Return empty list on parse error or empty result
		return []models.Drama{}, nil
	}

	var dramas []models.Drama
	for _, b := range raw.Items {
		// Search uses Title and Cover
		dramas = append(dramas, models.Drama{
			BookID:       "melolo:" + b.BookID,
			Judul:        b.Title,
			Cover:        p.proxyImage(b.Cover),
			Deskripsi:    b.Abstract,
			TotalEpisode: "0", // Search item might not have count
			Genre:        "",  // Search item might not have detailed tags
		})
	}
	return dramas, nil
}

func (p *MeloloProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// ID comes as "melolo:12345", strip prefix handled by manager usually, but here we just use rawID if passed correctly.
	// Manager passes rawID.

	url := fmt.Sprintf("%s/series?series_id=%s", MeloloAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	var raw mDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	// Parse Series Info
	drama := models.Drama{
		BookID:       "melolo:" + strconv.FormatInt(raw.Series.SeriesID, 10),
		Judul:        raw.Series.Title,
		Cover:        p.proxyImage(raw.Series.Cover),
		Deskripsi:    raw.Series.Intro,
		TotalEpisode: strconv.Itoa(raw.Series.EpisodeCount),
	}

	// Parse Episodes
	var episodes []models.Episode
	for _, ep := range raw.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       "melolo:" + strconv.FormatInt(raw.Series.SeriesID, 10),
			EpisodeIndex: int(ep.Index - 1), // API is 1-based
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Index),
			// We can store VID in a temporary map or just re-fetch logic in GetStream using array index matching
			// But specific GetStream video_id lookup requires knowing the ID.
			// We'll trust that Stream request will use the index to lookup again or we pass VID if architecture allows (it doesn't easily).
			// Wait, GetStream receives 'id' (bookID) and 'epIndex'.
			// We will re-fetch detail in GetStream to map index -> VID.
		})
	}

	return &drama, episodes, nil
}

func (p *MeloloProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// 1. Fetch Detail to map Index -> VID
	urlDetail := fmt.Sprintf("%s/series?series_id=%s", MeloloAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, err
	}

	var rawDetail mDetailResponse
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, err
	}

	idx, _ := strconv.Atoi(epIndex) // 0-based from frontend
	targetIndex := int64(idx + 1)   // 1-based API index

	var targetVid string
	for _, ep := range rawDetail.Episodes {
		if ep.Index == targetIndex {
			targetVid = ep.Vid
			break
		}
	}

	if targetVid == "" {
		return nil, fmt.Errorf("episode index %d not found", idx)
	}

	// 2. Fetch Video Stream
	urlStream := fmt.Sprintf("%s/video?video_id=%s", MeloloAPI, targetVid)
	bodyStream, err := p.fetch(urlStream)
	if err != nil {
		return nil, err
	}

	var rawStream mStreamResponse
	// Handle potential structure variations if needed, but per user request snippet:
	/*
		{
		  "video_id": "...",
		  "main_url": "http://..."
		}
	*/
	if err := json.Unmarshal(bodyStream, &rawStream); err != nil {
		return nil, err
	}

	if rawStream.MainURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}

	// Force HTTPS
	finalUrl := strings.Replace(rawStream.MainURL, "http://", "https://", 1)

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
