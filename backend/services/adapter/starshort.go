package adapter

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type StarshortProvider struct{}

const StarshortAPI = "https://sapimu.au/starshort/api/v1"

func NewStarshortProvider() *StarshortProvider {
	return &StarshortProvider{}
}

func (p *StarshortProvider) GetID() string {
	return "starshort"
}

func (p *StarshortProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *StarshortProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	// Headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer 0ebd6cfdd8054d2a90aa2851532645211aeaf189fa1aed62c53e5fd735af8649")

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

func (p *StarshortProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Internal Models ---

type sDrama struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Cover string   `json:"cover"`
	Tags  []string `json:"tags"`
}

type sDetail struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Cover         string   `json:"cover"`
	TotalEpisodes int      `json:"total_episodes"`
	FreeEpisodes  int      `json:"free_episodes"`
	Tags          []string `json:"tags"`
}

type sEpisodeResponse struct {
	DramaID  string     `json:"drama_id"`
	Title    string     `json:"title"`
	Total    int        `json:"total"`
	Episodes []sEpisode `json:"episodes"`
}

type sEpisode struct {
	Episode int  `json:"episode"`
	Free    bool `json:"free"`
	// Price field ignored
}

type sStreamResponse struct {
	VideoURL  string `json:"video_url"`
	ExpiresIn string `json:"expires_in"`
}

// --- Implementation ---

func (p *StarshortProvider) GetTrending() ([]models.Drama, error) {
	// Use /dramas/rising as trending
	body, err := p.fetch(StarshortAPI + "/dramas/rising?lang=3")
	if err != nil {
		return nil, err
	}

	var raw []sDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "starshort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
			Genre:  "", // Tags not joined here to match simple list, or could join d.Tags
		})
	}
	return dramas, nil
}

func (p *StarshortProvider) GetLatest(page int) ([]models.Drama, error) {
	// Use /dramas/new
	body, err := p.fetch(StarshortAPI + "/dramas/new?lang=3")
	if err != nil {
		return nil, err
	}

	var raw []sDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "starshort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *StarshortProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/dramas/search?q=%s&lang=3", StarshortAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw []sDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		// Return empty on error or empty body
		return []models.Drama{}, nil
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "starshort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *StarshortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// 1. Fetch Detail Info
	urlDetail := fmt.Sprintf("%s/dramas/%s?lang=3", StarshortAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}

	var rawDetail sDetail
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "starshort:" + rawDetail.ID,
		Judul:        rawDetail.Title,
		Cover:        p.proxyImage(rawDetail.Cover),
		Deskripsi:    rawDetail.Description,
		TotalEpisode: strconv.Itoa(rawDetail.TotalEpisodes),
		// Tags to Genre?
	}

	// 2. Fetch Episodes List
	urlEp := fmt.Sprintf("%s/dramas/%s/episodes?lang=3", StarshortAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return &drama, nil, nil // Return what we have if ep fail
	}

	var rawEp sEpisodeResponse
	if err := json.Unmarshal(bodyEp, &rawEp); err != nil {
		return &drama, nil, nil
	}

	var episodes []models.Episode
	for _, ep := range rawEp.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       "starshort:" + id,
			EpisodeIndex: ep.Episode - 1, // 0-based for internal logic
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}

	return &drama, episodes, nil
}

func (p *StarshortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1 // API uses 1-based episode number

	url := fmt.Sprintf("%s/dramas/%s/episodes/%d?lang=3", StarshortAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw sStreamResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	if raw.VideoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}

	return &models.StreamData{
		BookID: "starshort:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.VideoURL,
			},
		},
	}, nil
}
