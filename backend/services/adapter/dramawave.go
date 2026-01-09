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

type DramaWaveProvider struct{}

const DramaWaveAPI = "https://sapimu.au/dramawave/api/v1"

func NewDramaWaveProvider() *DramaWaveProvider {
	return &DramaWaveProvider{}
}

func (p *DramaWaveProvider) GetID() string {
	return "dramawave"
}

func (p *DramaWaveProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *DramaWaveProvider) fetch(targetURL string) ([]byte, error) {
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

func (p *DramaWaveProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

type dwDrama struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Cover string   `json:"cover"`
	Tags  []string `json:"tags"`
}

type dwDetail struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Cover         string   `json:"cover"`
	TotalEpisodes int      `json:"total_episodes"`
	FreeEpisodes  int      `json:"free_episodes"`
	Tags          []string `json:"tags"`
}

type dwEpisodeResponse struct {
	Episodes []dwEpisode `json:"episodes"`
}

type dwEpisode struct {
	Episode int  `json:"episode"`
	Free    bool `json:"free"`
}

type dwStreamResponse struct {
	VideoURL string `json:"video_url"`
}

func (p *DramaWaveProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(DramaWaveAPI + "/dramas/rising?lang=3")
	if err != nil {
		return nil, err
	}
	var raw []dwDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "dramawave:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *DramaWaveProvider) GetLatest(page int) ([]models.Drama, error) {
	body, err := p.fetch(DramaWaveAPI + "/dramas/new?lang=3")
	if err != nil {
		return nil, err
	}
	var raw []dwDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "dramawave:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *DramaWaveProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/dramas/search?q=%s&lang=3", DramaWaveAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}
	var raw []dwDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "dramawave:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *DramaWaveProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	urlDetail := fmt.Sprintf("%s/dramas/%s?lang=3", DramaWaveAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}
	var rawDetail dwDetail
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, nil, err
	}
	drama := models.Drama{
		BookID:       "dramawave:" + rawDetail.ID,
		Judul:        rawDetail.Title,
		Cover:        p.proxyImage(rawDetail.Cover),
		Deskripsi:    rawDetail.Description,
		TotalEpisode: strconv.Itoa(rawDetail.TotalEpisodes),
	}

	urlEp := fmt.Sprintf("%s/dramas/%s/episodes?lang=4", DramaWaveAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return &drama, nil, nil
	}
	var rawEp dwEpisodeResponse
	json.Unmarshal(bodyEp, &rawEp)
	var episodes []models.Episode
	for _, ep := range rawEp.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       "dramawave:" + id,
			EpisodeIndex: ep.Episode - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}
	return &drama, episodes, nil
}

func (p *DramaWaveProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1
	url := fmt.Sprintf("%s/dramas/%s/episodes/%d?lang=4", DramaWaveAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}
	var raw dwStreamResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if raw.VideoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}
	return &models.StreamData{
		BookID: "dramawave:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.VideoURL,
			},
		},
	}, nil
}
