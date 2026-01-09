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

type HiShortProvider struct{}

const HiShortAPI = "https://sapimu.au/hishort/api"

func NewHiShortProvider() *HiShortProvider {
	return &HiShortProvider{}
}

func (p *HiShortProvider) GetID() string {
	return "hishort"
}

func (p *HiShortProvider) IsCompatibleID(id string) bool {
	return true
}

// Specific token for HiShort provided by user
const HiShortToken = "0ebd6cfdd8054d2a90aa2851532645211aeaf189fa1aed62c53e5fd735af8649"

func (p *HiShortProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	// Use Specific HiShortToken instead of shared SapimuToken
	req.Header.Set("Authorization", "Bearer "+HiShortToken)
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

func (p *HiShortProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

type hsDetail struct {
	Title    string `json:"title"`
	Poster   string `json:"poster"`
	Synopsis string `json:"synopsis"`
	Episodes []struct {
		Slug   string `json:"slug"`
		Title  string `json:"title"`
		Number int    `json:"number"`
	} `json:"episodes"`
}

type hsStreamResponse struct {
	Title   string `json:"title"`
	Servers []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
		Type string `json:"type"`
	} `json:"servers"`
	Subtitles []struct {
		Lang string `json:"lang"`
		URL  string `json:"url"`
	} `json:"subtitles"`
}

type hsResponse struct {
	Popular []hsItem `json:"popular"`
}
type hsItem struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Poster string `json:"poster"`
	Type   string `json:"type"`
}

func (p *HiShortProvider) GetTrending() ([]models.Drama, error) {
	// Use /home for Trending
	body, err := p.fetch(HiShortAPI + "/home?lang=in")
	if err != nil {
		return nil, err
	}

	var raw hsResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Popular {
		dramas = append(dramas, models.Drama{
			BookID: "hishort:" + d.Slug,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Poster),
		})
	}
	return dramas, nil
}

func (p *HiShortProvider) GetLatest(page int) ([]models.Drama, error) {
	return p.GetTrending()
}

func (p *HiShortProvider) Search(query string) ([]models.Drama, error) {
	// Endpoint: /hishort/api/search/{query}?lang=in
	urlSearch := fmt.Sprintf("%s/search/%s?lang=in", HiShortAPI, url.QueryEscape(query))
	body, err := p.fetch(urlSearch)
	if err != nil {
		return nil, err
	}

	var raw []hsItem
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "hishort:" + d.Slug,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Poster),
		})
	}
	return dramas, nil
}

func (p *HiShortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// Endpoint: /hishort/api/drama/{id}?lang=in
	urlDetail := fmt.Sprintf("%s/drama/%s?lang=in", HiShortAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}

	var rawDetail hsDetail
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "hishort:" + id, // Slug is ID
		Judul:        rawDetail.Title,
		Cover:        p.proxyImage(rawDetail.Poster),
		Deskripsi:    rawDetail.Synopsis,
		TotalEpisode: strconv.Itoa(len(rawDetail.Episodes)),
	}

	var episodes []models.Episode
	for _, ep := range rawDetail.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       "hishort:" + id,
			EpisodeIndex: ep.Number - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Number),
		})
	}

	return &drama, episodes, nil
}

func (p *HiShortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1
	// Endpoint: /hishort/api/episode/{id}_{epNum}?lang=in
	// Note: HiShort uses underscore format for episode ID: "3688_1"
	url := fmt.Sprintf("%s/episode/%s_%d?lang=in", HiShortAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw hsStreamResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	videoURL := ""
	// Prioritize HLS
	for _, srv := range raw.Servers {
		if srv.Type == "hls" {
			videoURL = srv.URL
			break
		}
	}
	// Fallback first
	if videoURL == "" && len(raw.Servers) > 0 {
		videoURL = raw.Servers[0].URL
	}

	if videoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}

	return &models.StreamData{
		BookID: "hishort:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				M3u8: videoURL, // HiShort primarily returns HLS
			},
		},
	}, nil
}
