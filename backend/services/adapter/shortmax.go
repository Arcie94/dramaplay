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

type ShortMaxProvider struct{}

const ShortMaxAPI = "https://sapimu.au/shortmax/api/v1"

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

// Reusing same structs with `sm` prefix for safety
type smDrama struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Cover string   `json:"cover"`
	Tags  []string `json:"tags"`
}

type smDetail struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Cover         string   `json:"cover"`
	TotalEpisodes int      `json:"total_episodes"`
	FreeEpisodes  int      `json:"free_episodes"`
	Tags          []string `json:"tags"`
}

type smEpisodeResponse struct {
	DramaID  string      `json:"drama_id"`
	Title    string      `json:"title"`
	Total    int         `json:"total"`
	Episodes []smEpisode `json:"episodes"`
}

type smEpisode struct {
	Episode int  `json:"episode"`
	Free    bool `json:"free"`
}

type smStreamResponse struct {
	VideoURL  string `json:"video_url"`
	ExpiresIn string `json:"expires_in"`
}

func (p *ShortMaxProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(ShortMaxAPI + "/dramas/rising?lang=3")
	if err != nil {
		return nil, err
	}

	var raw []smDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "shortmax:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *ShortMaxProvider) GetLatest(page int) ([]models.Drama, error) {
	body, err := p.fetch(ShortMaxAPI + "/dramas/new?lang=3")
	if err != nil {
		return nil, err
	}

	var raw []smDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "shortmax:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *ShortMaxProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/dramas/search?q=%s&lang=3", ShortMaxAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw []smDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		return []models.Drama{}, nil
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "shortmax:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *ShortMaxProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	urlDetail := fmt.Sprintf("%s/dramas/%s?lang=3", ShortMaxAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}

	var rawDetail smDetail
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "shortmax:" + rawDetail.ID,
		Judul:        rawDetail.Title,
		Cover:        p.proxyImage(rawDetail.Cover),
		Deskripsi:    rawDetail.Description,
		TotalEpisode: strconv.Itoa(rawDetail.TotalEpisodes),
	}

	urlEp := fmt.Sprintf("%s/dramas/%s/episodes?lang=4", ShortMaxAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return &drama, nil, nil
	}

	var rawEp smEpisodeResponse
	if err := json.Unmarshal(bodyEp, &rawEp); err != nil {
		return &drama, nil, nil
	}

	var episodes []models.Episode
	for _, ep := range rawEp.Episodes {
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

	url := fmt.Sprintf("%s/dramas/%s/episodes/%d?lang=4", ShortMaxAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw smStreamResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	if raw.VideoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}

	return &models.StreamData{
		BookID: "shortmax:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.VideoURL,
			},
		},
	}, nil
}
