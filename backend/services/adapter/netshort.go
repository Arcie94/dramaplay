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
	"time"
)

type NetshortProvider struct {
	client *http.Client
}

const NetshortAPI = "https://api.sansekai.my.id/api/netshort"

func NewNetshortProvider() *NetshortProvider {
	return &NetshortProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *NetshortProvider) GetID() string {
	return "netshort"
}

func (p *NetshortProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *NetshortProvider) fetch(url string) ([]byte, error) {
	// Retry logic (3 times) with Exponential Backoff
	var lastErr error
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		// Spoof headers to look like a browser (Chrome Android)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Referer", "https://netshort.com/")

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 1 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)

			// If Rate Limit (429), wait longer
			if resp.StatusCode == 429 {
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
			} else {
				time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		return body, nil
	}
	return nil, lastErr
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
			EpisodeIndex: ep.EpisodeNo - 1,
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
	// Convert 0-based internal index to 1-based for Netshort lookup
	targetIndex := idx + 1

	var targetUrl string
	for _, ep := range raw.EpisodeInfos {
		if ep.EpisodeNo == targetIndex {
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
