package adapter

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DramaboxProvider struct {
	client *http.Client
}

const DramaboxAPI = "https://api.sansekai.my.id/api/dramabox"

func NewDramaboxProvider() *DramaboxProvider {
	return &DramaboxProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *DramaboxProvider) GetID() string {
	return "dramabox"
}

func (p *DramaboxProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *DramaboxProvider) fetch(url string) ([]byte, error) {
	// Retry logic (3 times)
	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := p.client.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond) // Backoff
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}

		return io.ReadAll(resp.Body)
	}
	return nil, lastErr
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

type SansekaiDetailResponse struct {
	Data struct {
		Book struct {
			BookID       string   `json:"bookId"`
			BookName     string   `json:"bookName"`
			Cover        string   `json:"cover"`
			Introduction string   `json:"introduction"`
			ChapterCount int      `json:"chapterCount"`
			Tags         []string `json:"tags"`
		} `json:"book"`
		ChapterList []struct {
			Index    int    `json:"index"`
			Mp4      string `json:"mp4"`
			M3u8Url  string `json:"m3u8Url"`
			Duration int    `json:"duration"`
		} `json:"chapterList"`
	} `json:"data"`
}

// --- Implementation ---

func (p *DramaboxProvider) GetTrending() ([]models.Drama, error) {
	body, err := p.fetch(DramaboxAPI + "/vip")
	if err != nil {
		return nil, err
	}

	// Sansekai API returns direct object with columnVoList array
	var vipData struct {
		ColumnVoList []struct {
			BookList []struct {
				BookID       string `json:"bookId"`
				BookName     string `json:"bookName"`
				CoverWap     string `json:"coverWap"`
				Introduction string `json:"introduction"`
				ChapterCount int    `json:"chapterCount"`
				TagV3s       []struct {
					TagName string `json:"tagName"`
				} `json:"tagV3s"`
			} `json:"bookList"`
		} `json:"columnVoList"`
	}

	if err := json.Unmarshal(body, &vipData); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	// Loop through all column sections and their books
	for _, column := range vipData.ColumnVoList {
		for _, b := range column.BookList {
			var tags []string
			for _, t := range b.TagV3s {
				tags = append(tags, t.TagName)
			}
			dramas = append(dramas, models.Drama{
				BookID:       "dramabox:" + b.BookID,
				Judul:        b.BookName,
				Cover:        b.CoverWap,
				Deskripsi:    b.Introduction,
				TotalEpisode: fmt.Sprintf("%d", b.ChapterCount),
				Genre:        strings.Join(tags, ", "),
			})
		}
	}
	return dramas, nil
}

func (p *DramaboxProvider) Search(query string) ([]models.Drama, error) {
	// Sansekai API doesn't have search endpoint yet
	// Fallback: get from /foryou and filter client-side
	body, err := p.fetch(DramaboxAPI + "/foryou")
	if err != nil {
		return nil, err
	}

	var books []struct {
		BookID       string `json:"bookId"`
		BookName     string `json:"bookName"`
		CoverWap     string `json:"coverWap"`
		Introduction string `json:"introduction"`
		ChapterCount int    `json:"chapterCount"`
		TagV3s       []struct {
			TagName string `json:"tagName"`
		} `json:"tagV3s"`
	}

	if err := json.Unmarshal(body, &books); err != nil {
		return nil, err
	}

	// Filter by query (case insensitive)
	query = strings.ToLower(query)
	var dramas []models.Drama
	for _, b := range books {
		if strings.Contains(strings.ToLower(b.BookName), query) ||
			strings.Contains(strings.ToLower(b.Introduction), query) {
			var tags []string
			for _, t := range b.TagV3s {
				tags = append(tags, t.TagName)
			}
			dramas = append(dramas, models.Drama{
				BookID:       "dramabox:" + b.BookID,
				Judul:        b.BookName,
				Cover:        b.CoverWap,
				Deskripsi:    b.Introduction,
				TotalEpisode: fmt.Sprintf("%d", b.ChapterCount),
				Genre:        strings.Join(tags, ", "),
			})
		}
	}
	return dramas, nil
}

func (p *DramaboxProvider) GetLatest(page int) ([]models.Drama, error) {
	// Sansekai API /foryou returns array directly, no pagination
	body, err := p.fetch(DramaboxAPI + "/foryou")
	if err != nil {
		return nil, err
	}

	var books []struct {
		BookID       string `json:"bookId"`
		BookName     string `json:"bookName"`
		CoverWap     string `json:"coverWap"`
		Introduction string `json:"introduction"`
		ChapterCount int    `json:"chapterCount"`
		TagV3s       []struct {
			TagName string `json:"tagName"`
		} `json:"tagV3s"`
	}

	if err := json.Unmarshal(body, &books); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range books {
		var tags []string
		for _, t := range b.TagV3s {
			tags = append(tags, t.TagName)
		}
		dramas = append(dramas, models.Drama{
			BookID:       "dramabox:" + b.BookID,
			Judul:        b.BookName,
			Cover:        b.CoverWap,
			Deskripsi:    b.Introduction,
			TotalEpisode: fmt.Sprintf("%d", b.ChapterCount),
			Genre:        strings.Join(tags, ", "),
		})
	}
	return dramas, nil
}

func (p *DramaboxProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	url := fmt.Sprintf("%s/detail?bookId=%s", DramaboxAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, nil, err
	}

	var raw SansekaiDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "dramabox:" + raw.Data.Book.BookID,
		Judul:        raw.Data.Book.BookName,
		Deskripsi:    raw.Data.Book.Introduction,
		Cover:        raw.Data.Book.Cover,
		TotalEpisode: fmt.Sprintf("%d", raw.Data.Book.ChapterCount),
		Genre:        strings.Join(raw.Data.Book.Tags, ", "),
	}

	var episodes []models.Episode
	for _, ch := range raw.Data.ChapterList {
		episodes = append(episodes, models.Episode{
			BookID:       "dramabox:" + raw.Data.Book.BookID,
			EpisodeIndex: ch.Index,
			EpisodeLabel: fmt.Sprintf("Episode %d", ch.Index+1),
		})
	}

	return &drama, episodes, nil
}

func (p *DramaboxProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// Use detail endpoint to get stream info (since it contains full chapter list with urls)
	url := fmt.Sprintf("%s/detail?bookId=%s", DramaboxAPI, id)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var raw SansekaiDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	idx, _ := strconv.Atoi(epIndex)
	for _, ch := range raw.Data.ChapterList {
		if ch.Index == idx {
			return &models.StreamData{
				BookID: "dramabox:" + id,
				Chapter: models.ChapterData{
					Index:    ch.Index,
					Duration: ch.Duration,
					Video: models.VideoData{
						Mp4:  ch.Mp4,
						M3u8: ch.M3u8Url,
					},
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("chapter index %d not found", idx)
}
