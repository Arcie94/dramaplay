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

type MeloloProvider struct{}

const MeloloAPI = "https://api.sansekai.my.id/api/melolo"

func NewMeloloProvider() *MeloloProvider {
	return &MeloloProvider{}
}

func (p *MeloloProvider) GetID() string {
	return "melolo"
}

func (p *MeloloProvider) IsCompatibleID(id string) bool {
	return true // Relies on Manager routing
}

func (p *MeloloProvider) fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (p *MeloloProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// --- Internal Models ---
type mListResponse struct {
	Books []mBook `json:"books"`
}
type mBook struct {
	BookID      string   `json:"book_id"`
	BookName    string   `json:"book_name"`
	ThumbURL    string   `json:"thumb_url"`
	Abstract    string   `json:"abstract"`
	SerialCount string   `json:"serial_count"`
	StatInfos   []string `json:"stat_infos"`
}
type mSearchResponse struct {
	Data struct {
		SearchData []struct {
			Books []mBook `json:"books"`
		} `json:"search_data"`
	} `json:"data"`
}
type mDetailResponse struct {
	Data struct {
		VideoList   []mVideoItem `json:"video_list"`
		SeriesTitle string       `json:"series_title"`
		SeriesIntro string       `json:"series_intro"`
		SeriesCover string       `json:"series_cover"`
		VideoData   *struct {
			VideoList   []mVideoItem `json:"video_list"`
			SeriesTitle string       `json:"series_title"`
			SeriesIntro string       `json:"series_intro"`
			SeriesCover string       `json:"series_cover"`
		} `json:"video_data"`
	} `json:"data"`
}
type mVideoItem struct {
	Vid      string `json:"vid"`
	Title    string `json:"title"`
	Cover    string `json:"cover"`
	Duration int    `json:"duration"`
	VidIndex int    `json:"vid_index"`
}
type mStreamResponse struct {
	Data struct {
		MainURL string `json:"main_url"`
	} `json:"data"`
}

// --- Implementation ---

func (p *MeloloProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(MeloloAPI + "/trending")
	if err != nil {
		return nil, err
	}

	var raw mListResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range raw.Books {
		dramas = append(dramas, models.Drama{
			BookID:       "melolo:" + b.BookID,
			Judul:        b.BookName,
			Cover:        p.proxyImage(b.ThumbURL),
			Deskripsi:    b.Abstract,
			TotalEpisode: b.SerialCount,
			Genre:        strings.Join(b.StatInfos, ", "),
		})
	}
	return dramas, nil
}

func (p *MeloloProvider) GetLatest(page int) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/latest?page=%d", MeloloAPI, page) // Assuming simple paging
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw mListResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range raw.Books {
		dramas = append(dramas, models.Drama{
			BookID:       "melolo:" + b.BookID,
			Judul:        b.BookName,
			Cover:        p.proxyImage(b.ThumbURL),
			Deskripsi:    b.Abstract,
			TotalEpisode: b.SerialCount,
			Genre:        strings.Join(b.StatInfos, ", "),
		})
	}
	return dramas, nil
}

func (p *MeloloProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/search?query=%s", MeloloAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw mSearchResponse
	json.Unmarshal(body, &raw)

	var dramas []models.Drama
	for _, group := range raw.Data.SearchData {
		for _, b := range group.Books {
			dramas = append(dramas, models.Drama{
				BookID:       "melolo:" + b.BookID,
				Judul:        b.BookName,
				Cover:        p.proxyImage(b.ThumbURL),
				Deskripsi:    b.Abstract,
				TotalEpisode: b.SerialCount,
				Genre:        strings.Join(b.StatInfos, ", "),
			})
		}
	}
	return dramas, nil
}

func (p *MeloloProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	url := fmt.Sprintf("%s/detail?bookId=%s", MeloloAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	var raw mDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	var title, cover, intro string
	var videoList []mVideoItem

	if raw.Data.VideoData != nil && len(raw.Data.VideoData.VideoList) > 0 {
		title = raw.Data.VideoData.SeriesTitle
		cover = raw.Data.VideoData.SeriesCover
		intro = raw.Data.VideoData.SeriesIntro
		videoList = raw.Data.VideoData.VideoList
	} else {
		title = raw.Data.SeriesTitle
		cover = raw.Data.SeriesCover
		intro = raw.Data.SeriesIntro
		videoList = raw.Data.VideoList
	}

	drama := models.Drama{
		BookID:       "melolo:" + id,
		Judul:        title,
		Cover:        p.proxyImage(cover),
		Deskripsi:    intro,
		TotalEpisode: strconv.Itoa(len(videoList)),
	}

	var episodes []models.Episode
	for _, v := range videoList {
		episodes = append(episodes, models.Episode{
			BookID:       "melolo:" + id,
			EpisodeIndex: v.VidIndex - 1, // Normalize to 0-based
			EpisodeLabel: fmt.Sprintf("Episode %d", v.VidIndex),
		})
	}

	return &drama, episodes, nil
}

func (p *MeloloProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	fmt.Printf("[Melolo] GetStream Called. ID: %s, Index: %s\n", id, epIndex)

	// 1. Need Detail to map Index -> VID
	urlDetail := fmt.Sprintf("%s/detail?bookId=%s", MeloloAPI, id)
	bodyDetail, err := p.fetch(urlDetail)
	if err != nil {
		fmt.Printf("[Melolo] Detail fetch failed: %v\n", err)
		return nil, err
	}
	var rawDetail mDetailResponse
	json.Unmarshal(bodyDetail, &rawDetail)

	var videoList []mVideoItem
	if rawDetail.Data.VideoData != nil && len(rawDetail.Data.VideoData.VideoList) > 0 {
		videoList = rawDetail.Data.VideoData.VideoList
	} else {
		videoList = rawDetail.Data.VideoList
	}

	idx, _ := strconv.Atoi(epIndex)
	var targetVid string

	// Melolo VidIndex is 1-based (usually)
	// We receive 0-based index from frontend
	targetVidIndex := idx + 1

	for _, ep := range videoList {
		if ep.VidIndex == targetVidIndex {
			targetVid = ep.Vid
			break
		}
	}

	fmt.Printf("[Melolo] Looking for Index %d (VidIndex %d). Found VID: %s\n", idx, targetVidIndex, targetVid)

	if targetVid == "" {
		// Fallback: Use array index directly if within bounds
		// This handles cases where VidIndex might be missing or non-sequential
		if idx >= 0 && idx < len(videoList) {
			targetVid = videoList[idx].Vid
			fmt.Printf("[Melolo] Fallback to Array Index [%d]. VID: %s\n", idx, targetVid)
		} else {
			fmt.Printf("[Melolo] Episode not found for index %d\n", idx)
			return nil, fmt.Errorf("episode not found")
		}
	}

	// 2. Fetch Stream
	urlStream := fmt.Sprintf("%s/stream?videoId=%s", MeloloAPI, targetVid)
	fmt.Printf("[Melolo] Fetching stream from: %s\n", urlStream)

	bodyStream, err := p.fetch(urlStream)
	if err != nil {
		fmt.Printf("[Melolo] Stream fetch failed: %v\n", err)
		return nil, err
	}
	var rawStream mStreamResponse
	json.Unmarshal(bodyStream, &rawStream)

	if rawStream.Data.MainURL == "" {
		fmt.Printf("[Melolo] Stream URL empty in response: %s\n", string(bodyStream))
		return nil, fmt.Errorf("stream url empty")
	}

	streamURL := strings.Replace(rawStream.Data.MainURL, "http://", "https://", 1)

	return &models.StreamData{
		BookID: "melolo:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: streamURL,
			},
		},
	}, nil
}
