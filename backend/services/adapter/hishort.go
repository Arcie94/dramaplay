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

func (p *HiShortProvider) fetch(targetURL string) ([]byte, error) {
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

func (p *HiShortProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

type hsDrama struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Cover string   `json:"cover"`
	Tags  []string `json:"tags"`
}

type hsDetail struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Cover         string   `json:"cover"`
	TotalEpisodes int      `json:"total_episodes"`
	FreeEpisodes  int      `json:"free_episodes"`
	Tags          []string `json:"tags"`
}

type hsEpisodeResponse struct {
	Episodes []hsEpisode `json:"episodes"`
}

type hsEpisode struct {
	Episode int  `json:"episode"`
	Free    bool `json:"free"`
}

type hsStreamResponse struct {
	VideoURL string `json:"video_url"`
}

func (p *HiShortProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(HiShortAPI + "/dramas/rising?lang=3")
	if err != nil {
		return nil, err
	}
	var raw []hsDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "hishort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *HiShortProvider) GetLatest(page int) ([]models.Drama, error) {
	body, err := p.fetch(HiShortAPI + "/dramas/new?lang=3")
	if err != nil {
		return nil, err
	}
	var raw []hsDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "hishort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *HiShortProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/dramas/search?q=%s&lang=3", HiShortAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}
	var raw []hsDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "hishort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *HiShortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	urlDetail := fmt.Sprintf("%s/dramas/%s?lang=3", HiShortAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}
	var rawDetail hsDetail
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, nil, err
	}
	drama := models.Drama{
		BookID:       "hishort:" + rawDetail.ID,
		Judul:        rawDetail.Title,
		Cover:        p.proxyImage(rawDetail.Cover),
		Deskripsi:    rawDetail.Description,
		TotalEpisode: strconv.Itoa(rawDetail.TotalEpisodes),
	}

	urlEp := fmt.Sprintf("%s/dramas/%s/episodes?lang=4", HiShortAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return &drama, nil, nil
	}
	var rawEp hsEpisodeResponse
	json.Unmarshal(bodyEp, &rawEp)
	var episodes []models.Episode
	for _, ep := range rawEp.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       "hishort:" + id,
			EpisodeIndex: ep.Episode - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}
	return &drama, episodes, nil
}

func (p *HiShortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1
	url := fmt.Sprintf("%s/dramas/%s/episodes/%d?lang=4", HiShortAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}
	var raw hsStreamResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if raw.VideoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}
	return &models.StreamData{
		BookID: "hishort:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.VideoURL,
			},
		},
	}, nil
}
