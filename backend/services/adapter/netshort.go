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

type NetshortProvider struct{}

const NetshortAPI = "https://api.sansekai.my.id/api/netshort"

func NewNetshortProvider() *NetshortProvider {
	return &NetshortProvider{}
}

func (p *NetshortProvider) GetID() string {
	return "netshort"
}

func (p *NetshortProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *NetshortProvider) fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// --- Netshort Models ---
type nsHomeResponse struct {
	ContentInfos []nsBook `json:"contentInfos"`
}
type nsSearchResponse struct {
	SearchCodeSearchResult []nsBook `json:"searchCodeSearchResult"`
}
type nsBook struct {
	ShortPlayId    string `json:"shortPlayId"`
	ShortPlayName  string `json:"shortPlayName"`
	ShortPlayCover string `json:"shortPlayCover"`
	// Intro might be missing in list
}
type nsDetailResponse struct {
	ShortPlayId    string      `json:"shortPlayId"`
	ShortPlayName  string      `json:"shortPlayName"`
	ShortPlayCover string      `json:"shortPlayCover"`
	ShotIntroduce  string      `json:"shotIntroduce"`
	TotalEpisode   int         `json:"totalEpisode"`
	EpisodeInfos   []nsEpisode `json:"shortPlayEpisodeInfos"`
}
type nsEpisode struct {
	EpisodeId   string `json:"episodeId"`
	EpisodeNo   int    `json:"episodeNo"`
	PlayVoucher string `json:"playVoucher"`
	// Ignore subtitles for now
}

// --- Implementation ---

func (p *NetshortProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(NetshortAPI + "/foryou")
	if err != nil {
		return nil, err
	}

	var raw nsHomeResponse
	err = json.Unmarshal(body, &raw)
	// Some endpoints imply different structure? Checking specs...
	// /foryou returns object with contentInfos array. Correct.
	if err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range raw.ContentInfos {
		dramas = append(dramas, models.Drama{
			BookID:       "netshort:" + b.ShortPlayId,
			Judul:        b.ShortPlayName,
			Cover:        b.ShortPlayCover, // Netshort covers seem accessible directly?
			Deskripsi:    "",               // No desc in list
			TotalEpisode: "",
			Genre:        "Netshort",
		})
	}
	return dramas, nil
}

func (p *NetshortProvider) GetLatest(page int) ([]models.Drama, error) {
	// Fallback to Trending for now as we don't have a clear "Latest" endpoint with paging
	return p.GetTrending()
}

func (p *NetshortProvider) Search(query string) ([]models.Drama, error) {
	// Netshort prefers %20 over +
	encodedQuery := url.QueryEscape(query)
	encodedQuery = strings.ReplaceAll(encodedQuery, "+", "%20")
	url := fmt.Sprintf("%s/search?query=%s", NetshortAPI, encodedQuery)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw nsSearchResponse
	json.Unmarshal(body, &raw)

	var dramas []models.Drama
	for _, b := range raw.SearchCodeSearchResult {
		dramas = append(dramas, models.Drama{
			BookID:    "netshort:" + b.ShortPlayId,
			Judul:     b.ShortPlayName,
			Cover:     b.ShortPlayCover,
			Deskripsi: "",
			Genre:     "Netshort",
		})
	}
	return dramas, nil
}

func (p *NetshortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	url := fmt.Sprintf("%s/allepisode?shortPlayId=%s", NetshortAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	var raw nsDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "netshort:" + raw.ShortPlayId,
		Judul:        raw.ShortPlayName,
		Cover:        raw.ShortPlayCover,
		Deskripsi:    raw.ShotIntroduce,
		TotalEpisode: strconv.Itoa(raw.TotalEpisode),
	}

	var episodes []models.Episode
	for _, ep := range raw.EpisodeInfos {
		episodes = append(episodes, models.Episode{
			BookID:       "netshort:" + raw.ShortPlayId,
			EpisodeIndex: ep.EpisodeNo,
			EpisodeLabel: fmt.Sprintf("Episode %d", ep.EpisodeNo),
		})
	}

	return &drama, episodes, nil
}

func (p *NetshortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// Must fetch detail to get playVoucher
	url := fmt.Sprintf("%s/allepisode?shortPlayId=%s", NetshortAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw nsDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	idx, _ := strconv.Atoi(epIndex)
	var targetUrl string
	for _, ep := range raw.EpisodeInfos {
		if ep.EpisodeNo == idx {
			targetUrl = ep.PlayVoucher
			break
		}
	}

	if targetUrl == "" {
		return nil, fmt.Errorf("episode not found")
	}

	// Netshort playVoucher is a direct MP4 link (or redirect)
	return &models.StreamData{
		BookID: "netshort:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: targetUrl,
			},
		},
	}, nil
}
