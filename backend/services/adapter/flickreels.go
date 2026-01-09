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

type FlickReelsProvider struct{}

const FlickReelsAPI = "https://sapimu.au/flickreels/api/v1"

func NewFlickReelsProvider() *FlickReelsProvider {
	return &FlickReelsProvider{}
}

func (p *FlickReelsProvider) GetID() string {
	return "flickreels"
}

func (p *FlickReelsProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *FlickReelsProvider) fetch(targetURL string) ([]byte, error) {
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

func (p *FlickReelsProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

type frDrama struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Cover string   `json:"cover"`
	Tags  []string `json:"tags"`
}

type frDetail struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Cover         string   `json:"cover"`
	TotalEpisodes int      `json:"total_episodes"`
	FreeEpisodes  int      `json:"free_episodes"`
	Tags          []string `json:"tags"`
}

type frEpisodeResponse struct {
	Episodes []frEpisode `json:"episodes"`
}

type frEpisode struct {
	Episode int  `json:"episode"`
	Free    bool `json:"free"`
}

type frStreamResponse struct {
	VideoURL string `json:"video_url"`
}

func (p *FlickReelsProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(FlickReelsAPI + "/dramas/rising?lang=4")
	if err != nil {
		return nil, err
	}
	var raw []frDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "flickreels:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *FlickReelsProvider) GetLatest(page int) ([]models.Drama, error) {
	body, err := p.fetch(FlickReelsAPI + "/dramas/new?lang=4")
	if err != nil {
		return nil, err
	}
	var raw []frDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "flickreels:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *FlickReelsProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/dramas/search?q=%s&lang=4", FlickReelsAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}
	var raw []frDrama
	json.Unmarshal(body, &raw)
	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "flickreels:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *FlickReelsProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	urlDetail := fmt.Sprintf("%s/dramas/%s?lang=4", FlickReelsAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}
	var rawDetail frDetail
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, nil, err
	}
	drama := models.Drama{
		BookID:       "flickreels:" + rawDetail.ID,
		Judul:        rawDetail.Title,
		Cover:        p.proxyImage(rawDetail.Cover),
		Deskripsi:    rawDetail.Description,
		TotalEpisode: strconv.Itoa(rawDetail.TotalEpisodes),
	}

	urlEp := fmt.Sprintf("%s/dramas/%s/episodes?lang=4", FlickReelsAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return &drama, nil, nil
	}
	var rawEp frEpisodeResponse
	json.Unmarshal(bodyEp, &rawEp)
	var episodes []models.Episode
	for _, ep := range rawEp.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       "flickreels:" + id,
			EpisodeIndex: ep.Episode - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}
	return &drama, episodes, nil
}

func (p *FlickReelsProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1
	url := fmt.Sprintf("%s/dramas/%s/episodes/%d?lang=4", FlickReelsAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}
	var raw frStreamResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if raw.VideoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}
	return &models.StreamData{
		BookID: "flickreels:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.VideoURL,
			},
		},
	}, nil
}
