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

type DramaDashProvider struct{}

const DramaDashAPI = "https://sapimu.au/dramadash" // Assuming standardized endpoints even on root path based on pattern

func NewDramaDashProvider() *DramaDashProvider {
	return &DramaDashProvider{}
}

func (p *DramaDashProvider) GetID() string {
	return "dramadash"
}

func (p *DramaDashProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *DramaDashProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
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

func (p *DramaDashProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

type ddDrama struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Cover string   `json:"cover"`
	Tags  []string `json:"tags"`
}

type ddDetail struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Cover         string   `json:"cover"`
	TotalEpisodes int      `json:"total_episodes"`
	FreeEpisodes  int      `json:"free_episodes"`
	Tags          []string `json:"tags"`
}

type ddEpisodeResponse struct {
	Episodes []ddEpisode `json:"episodes"`
}

type ddEpisode struct {
	Episode int  `json:"episode"`
	Free    bool `json:"free"`
}

type ddStreamResponse struct {
	VideoURL string `json:"video_url"`
}

func (p *DramaDashProvider) GetTrending() ([]models.Drama, error) {
	// DramaDash might differ on sub-paths? Assuming standard /dramas/rising
	body, err := p.fetch(DramaDashAPI + "/dramas/rising?lang=4")
	if err != nil {
		return nil, err
	}
	var raw []ddDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "dramadash:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *DramaDashProvider) GetLatest(page int) ([]models.Drama, error) {
	body, err := p.fetch(DramaDashAPI + "/dramas/new?lang=4")
	if err != nil {
		return nil, err
	}
	var raw []ddDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "dramadash:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *DramaDashProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/dramas/search?q=%s&lang=4", DramaDashAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}
	var raw []ddDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "dramadash:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *DramaDashProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	urlDetail := fmt.Sprintf("%s/dramas/%s?lang=4", DramaDashAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}
	var rawDetail ddDetail
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, nil, err
	}
	drama := models.Drama{
		BookID:       "dramadash:" + rawDetail.ID,
		Judul:        rawDetail.Title,
		Cover:        p.proxyImage(rawDetail.Cover),
		Deskripsi:    rawDetail.Description,
		TotalEpisode: strconv.Itoa(rawDetail.TotalEpisodes),
	}

	urlEp := fmt.Sprintf("%s/dramas/%s/episodes?lang=4", DramaDashAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return &drama, nil, nil
	}
	var rawEp ddEpisodeResponse
	json.Unmarshal(bodyEp, &rawEp)
	var episodes []models.Episode
	for _, ep := range rawEp.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       "dramadash:" + id,
			EpisodeIndex: ep.Episode - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}
	return &drama, episodes, nil
}

func (p *DramaDashProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1
	url := fmt.Sprintf("%s/dramas/%s/episodes/%d?lang=4", DramaDashAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}
	var raw ddStreamResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if raw.VideoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}
	return &models.StreamData{
		BookID: "dramadash:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.VideoURL,
			},
		},
	}, nil
}
