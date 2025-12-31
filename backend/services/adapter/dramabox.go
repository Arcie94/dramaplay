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

type DramaboxProvider struct{}

const DramaboxAPI = "https://dramabox-api-rho.vercel.app/api"

func NewDramaboxProvider() *DramaboxProvider {
	return &DramaboxProvider{}
}

func (p *DramaboxProvider) GetID() string {
	return "dramabox"
}

func (p *DramaboxProvider) IsCompatibleID(id string) bool {
	// IDs from this provider should be prefixed with "dramabox:" in the app
	// But this method might be used to check raw IDs if needed.
	// Primary routing is done by checking the prefix in the Manager.
	// Here we can check if it looks like a Dramabox ID (usually UUID or numeric string < 15 chars)
	// For robust routing, we rely on the Manager's prefix check.
	return true
}

func (p *DramaboxProvider) fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// --- Ext Models (internal to this provider) ---
type dbResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}
type dbHomeData struct {
	Book []dbBook `json:"book"`
}
type dbSearchData struct {
	Book []dbBookSearch `json:"book"`
}
type dbBook struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Cover        string  `json:"cover"`
	Introduction string  `json:"introduction"`
	ChapterCount int     `json:"chapterCount"`
	Tags         []dbTag `json:"tags"`
}
type dbBookSearch struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Cover        string   `json:"cover"`
	Introduction string   `json:"introduction"`
	Tags         []string `json:"tags"`
}
type dbTag struct {
	TagName string `json:"tagName"`
}
type dbDetailData struct {
	Drama    dbDramaDetail `json:"drama"`
	Chapters []dbChapter   `json:"chapters"`
}
type dbDramaDetail struct {
	BookID       string `json:"bookId"`
	BookName     string `json:"bookName"`
	Cover        string `json:"cover"`
	Introduction string `json:"introduction"`
	ChapterCount int    `json:"chapterCount"`
}
type dbChapter struct {
	Index int `json:"index"`
}
type dbStreamResponse struct {
	Data struct {
		Chapter struct {
			Index int `json:"index"`
			Video struct {
				Mp4  string `json:"mp4"`
				M3u8 string `json:"m3u8"`
			} `json:"video"`
			Duration int `json:"duration"`
		} `json:"chapter"`
	} `json:"data"`
}

type ExtCategoryResponse struct {
	Success bool            `json:"success"`
	Data    ExtCategoryData `json:"data"`
}
type ExtCategoryData struct {
	BookList    []dbBookCategory `json:"bookList"`
	CurrentPage int              `json:"currentPage"`
	Total       int64            `json:"total"`
}
type dbBookCategory struct {
	BookID       string   `json:"bookId"`
	BookName     string   `json:"bookName"`
	Cover        string   `json:"cover"`
	Introduction string   `json:"introduction"`
	ChapterCount int      `json:"chapterCount"`
	Tags         []string `json:"tags"`
}

// --- Implementation ---

func (p *DramaboxProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(DramaboxAPI + "/home")
	if err != nil {
		return nil, err
	}

	var raw dbResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var homeData dbHomeData
	json.Unmarshal(raw.Data, &homeData)

	var dramas []models.Drama
	for _, b := range homeData.Book {
		var tags []string
		for _, t := range b.Tags {
			tags = append(tags, t.TagName)
		}
		dramas = append(dramas, models.Drama{
			BookID:       "dramabox:" + b.ID, // Prefixing ID
			Judul:        b.Name,
			Cover:        b.Cover,
			Deskripsi:    b.Introduction,
			TotalEpisode: fmt.Sprintf("%d", b.ChapterCount),
			Genre:        strings.Join(tags, ", "),
		})
	}
	return dramas, nil
}

func (p *DramaboxProvider) Search(query string) ([]models.Drama, error) {
	url := fmt.Sprintf("%s/search?keyword=%s", DramaboxAPI, url.QueryEscape(query))
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw dbResponse
	json.Unmarshal(body, &raw)
	var searchData dbSearchData
	json.Unmarshal(raw.Data, &searchData)

	var dramas []models.Drama
	for _, b := range searchData.Book {
		dramas = append(dramas, models.Drama{
			BookID:    "dramabox:" + b.ID, // Prefixing ID
			Judul:     b.Name,
			Cover:     b.Cover,
			Deskripsi: b.Introduction,
			Genre:     strings.Join(b.Tags, ", "),
		})
	}
	return dramas, nil
}

func (p *DramaboxProvider) GetLatest(page int) ([]models.Drama, error) {
	// Category 0 is "All" / Latest
	url := fmt.Sprintf("%s/category/0?page=%d", DramaboxAPI, page)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw ExtCategoryResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range raw.Data.BookList {
		dramas = append(dramas, models.Drama{
			BookID:       "dramabox:" + b.BookID,
			Judul:        b.BookName,
			Cover:        b.Cover,
			Deskripsi:    b.Introduction,
			TotalEpisode: fmt.Sprintf("%d", b.ChapterCount),
			Genre:        strings.Join(b.Tags, ", "),
		})
	}
	return dramas, nil
}

func (p *DramaboxProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// ID passed here is already stripped of prefix by Manager
	url := fmt.Sprintf("%s/detail/%s/v2", DramaboxAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	var raw dbResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}
	var detailData dbDetailData
	if err := json.Unmarshal(raw.Data, &detailData); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "dramabox:" + detailData.Drama.BookID,
		Judul:        detailData.Drama.BookName,
		Deskripsi:    detailData.Drama.Introduction,
		Cover:        detailData.Drama.Cover,
		TotalEpisode: fmt.Sprintf("%d", detailData.Drama.ChapterCount),
	}

	var episodes []models.Episode
	for _, ch := range detailData.Chapters {
		episodes = append(episodes, models.Episode{
			BookID:       "dramabox:" + detailData.Drama.BookID,
			EpisodeIndex: ch.Index + 1,
			EpisodeLabel: fmt.Sprintf("Episode %d", ch.Index+1),
		})
	}

	return &drama, episodes, nil
}

func (p *DramaboxProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	idx, _ := strconv.Atoi(epIndex)
	// Convert 0-based internal index to 1-based for Dramabox API
	upstreamIndex := idx + 1

	url := fmt.Sprintf("%s/stream?bookId=%s&episode=%d", DramaboxAPI, id, upstreamIndex)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var streamResp dbStreamResponse
	if err := json.Unmarshal(body, &streamResp); err != nil {
		return nil, err
	}

	// Just return standard StreamData
	return &models.StreamData{
		BookID: "dramabox:" + id,
		Chapter: models.ChapterData{
			Index:    streamResp.Data.Chapter.Index,
			Duration: streamResp.Data.Chapter.Duration,
			Video: models.VideoData{
				Mp4:  streamResp.Data.Chapter.Video.Mp4,
				M3u8: streamResp.Data.Chapter.Video.M3u8,
			},
		},
	}, nil
}
