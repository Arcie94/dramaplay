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

type FreeShortProvider struct{}

const FreeShortAPI = "https://sapimu.au/freeshort/api/v1"

func NewFreeShortProvider() *FreeShortProvider {
	return &FreeShortProvider{}
}

func (p *FreeShortProvider) GetID() string {
	return "freeshort"
}

func (p *FreeShortProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *FreeShortProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
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

func (p *FreeShortProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	// Use wsrv.nl for simple proxy/resize
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// Reuse structs from Starshort if structure is identical (likely)
// But Go doesn't like cross-file private structs easily unless exported.
// I will copy private structs to be safe and independent.

type fsDrama struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Cover string   `json:"cover"`
	Tags  []string `json:"tags"`
}

type fsDetail struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Cover         string   `json:"cover"`
	TotalEpisodes int      `json:"total_episodes"`
	FreeEpisodes  int      `json:"free_episodes"`
	Tags          []string `json:"tags"`
}

type fsEpisodeResponse struct {
	DramaID  string      `json:"drama_id"`
	Title    string      `json:"title"`
	Total    int         `json:"total"`
	Episodes []fsEpisode `json:"episodes"`
}

type fsEpisode struct {
	Episode int  `json:"episode"`
	Free    bool `json:"free"`
}

type fsStreamResponse struct {
	VideoURL  string `json:"video_url"`
	ExpiresIn string `json:"expires_in"`
}

func (p *FreeShortProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(FreeShortAPI + "/dramas/rising?lang=4")
	if err != nil {
		return nil, err
	}

	var raw []fsDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "freeshort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *FreeShortProvider) GetLatest(page int) ([]models.Drama, error) {
	body, err := p.fetch(FreeShortAPI + "/dramas/new?lang=4")
	if err != nil {
		return nil, err
	}

	var raw []fsDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "freeshort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *FreeShortProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/dramas/search?q=%s&lang=4", FreeShortAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw []fsDrama
	if err := json.Unmarshal(body, &raw); err != nil {
		return []models.Drama{}, nil
	}

	var dramas []models.Drama
	for _, d := range raw {
		dramas = append(dramas, models.Drama{
			BookID: "freeshort:" + d.ID,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
		})
	}
	return dramas, nil
}

func (p *FreeShortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	urlDetail := fmt.Sprintf("%s/dramas/%s?lang=4", FreeShortAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}

	var rawDetail fsDetail
	if err := json.Unmarshal(bodyDetail, &rawDetail); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "freeshort:" + rawDetail.ID,
		Judul:        rawDetail.Title,
		Cover:        p.proxyImage(rawDetail.Cover),
		Deskripsi:    rawDetail.Description,
		TotalEpisode: strconv.Itoa(rawDetail.TotalEpisodes),
	}

	urlEp := fmt.Sprintf("%s/dramas/%s/episodes?lang=4", FreeShortAPI, id)
	bodyEp, err := p.fetch(urlEp)
	if err != nil {
		return &drama, nil, nil
	}

	var rawEp fsEpisodeResponse
	if err := json.Unmarshal(bodyEp, &rawEp); err != nil {
		return &drama, nil, nil
	}

	var episodes []models.Episode
	for _, ep := range rawEp.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       "freeshort:" + id,
			EpisodeIndex: ep.Episode - 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.Episode),
		})
	}

	return &drama, episodes, nil
}

func (p *FreeShortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	epNum := idx + 1

	url := fmt.Sprintf("%s/dramas/%s/episodes/%d?lang=4", FreeShortAPI, id, epNum)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw fsStreamResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	if raw.VideoURL == "" {
		return nil, fmt.Errorf("stream url empty")
	}

	return &models.StreamData{
		BookID: "freeshort:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: raw.VideoURL,
			},
		},
	}, nil
}
