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

const DramaboxAPI = "https://dramabos.asia/api/dramabox"

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
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		// Browser-like headers for dramabos.asia
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
		req.Header.Set("Referer", "https://dramabos.asia/")
		req.Header.Set("Origin", "https://dramabos.asia")

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 1 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			time.Sleep(time.Duration(i+1) * 1 * time.Second)
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

// Helper to proxy images
func (p *DramaboxProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	// Use wsrv.nl for caching, resizing and CORS proxy
	return "https://wsrv.nl/?url=" + strings.ReplaceAll(originalURL, "https://", "") + "&output=jpg"
}

// --- Internal Models ---

type dbResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

type dbBookList struct {
	List []dbBook `json:"list"`
}

type dbBook struct {
	BookID       string `json:"bookId"`
	BookName     string `json:"bookName"`
	Cover        string `json:"cover"`
	Introduction string `json:"introduction"`
}

type dbSearchList struct {
	List []dbBook `json:"searchResult"`
}

type dbDetailData struct {
	BookID       string `json:"bookId"`
	BookName     string `json:"bookName"`
	Cover        string `json:"cover"`
	Introduction string `json:"introduction"`
}

type dbChapterList struct {
	ChapterList []dbChapter `json:"chapterList"`
}

type dbChapter struct {
	ChapterID    string `json:"chapterId"`
	ChapterIndex int    `json:"chapterIndex"`
}

type dbPlayerData struct {
	BookID       string `json:"bookId"`
	ChapterIndex int    `json:"chapterIndex"`
	VideoURL     string `json:"videoUrl"`
}

// --- Implementation ---

func (p *DramaboxProvider) GetTrending() ([]models.Drama, error) {
	// Endpoint: /foryou/1
	body, err := p.fetch(DramaboxAPI + "/foryou/1")
	if err != nil {
		return nil, err
	}

	var resp dbResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("api returned success=false")
	}

	var data dbBookList
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range data.List {
		dramas = append(dramas, models.Drama{
			BookID:    "dramabox:" + b.BookID,
			Judul:     b.BookName,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Introduction,
		})
	}
	return dramas, nil
}

func (p *DramaboxProvider) GetLatest(page int) ([]models.Drama, error) {
	// Endpoint: /new/{page}
	if page < 1 {
		page = 1
	}
	url := fmt.Sprintf("%s/new/%d", DramaboxAPI, page)
	body, err := p.fetch(url)
	if err != nil {
		return nil, err
	}

	var resp dbResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var data dbBookList
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, b := range data.List {
		dramas = append(dramas, models.Drama{
			BookID:    "dramabox:" + b.BookID,
			Judul:     b.BookName,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Introduction,
		})
	}
	return dramas, nil
}

func (p *DramaboxProvider) Search(query string) ([]models.Drama, error) {
	// Endpoint: /search/{query}/1
	urlSearch := fmt.Sprintf("%s/search/%s/1", DramaboxAPI, query)

	body, err := p.fetch(urlSearch)
	if err != nil {
		return nil, err
	}

	var resp dbResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var data dbBookList
	// Try 'list' key first
	if err := json.Unmarshal(resp.Data, &data); err == nil && len(data.List) > 0 {
		// Found it
	} else {
		// Try 'searchResult' key
		var searchData dbSearchList
		if err2 := json.Unmarshal(resp.Data, &searchData); err2 == nil {
			data.List = searchData.List
		}
	}

	var dramas []models.Drama
	for _, b := range data.List {
		dramas = append(dramas, models.Drama{
			BookID:    "dramabox:" + b.BookID,
			Judul:     b.BookName,
			Cover:     p.proxyImage(b.Cover),
			Deskripsi: b.Introduction,
		})
	}
	return dramas, nil
}

func (p *DramaboxProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// Endpoint: /drama/{id}?lang=in
	urlDetail := fmt.Sprintf("%s/drama/%s?lang=in", DramaboxAPI, id)
	body, err := p.fetch(urlDetail)
	if err != nil {
		return nil, nil, err
	}

	var resp dbResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, err
	}

	var detail dbDetailData
	if err := json.Unmarshal(resp.Data, &detail); err != nil {
		return nil, nil, err
	}

	// Fetch Chapters: /chapters/{id}
	urlChapters := fmt.Sprintf("%s/chapters/%s", DramaboxAPI, id)
	bodyChap, err := p.fetch(urlChapters)
	if err != nil {
		return nil, nil, err
	}

	var respChap dbResponse
	if err := json.Unmarshal(bodyChap, &respChap); err != nil {
		return nil, nil, err
	}

	var chapData dbChapterList
	if err := json.Unmarshal(respChap.Data, &chapData); err != nil {
		return nil, nil, err
	}

	drama := models.Drama{
		BookID:       "dramabox:" + detail.BookID,
		Judul:        detail.BookName,
		Cover:        p.proxyImage(detail.Cover),
		Deskripsi:    detail.Introduction,
		TotalEpisode: strconv.Itoa(len(chapData.ChapterList)),
	}

	var episodes []models.Episode
	for _, ch := range chapData.ChapterList {
		episodes = append(episodes, models.Episode{
			BookID:       "dramabox:" + detail.BookID,
			EpisodeIndex: ch.ChapterIndex,
			EpisodeLabel: fmt.Sprintf("Episode %d", ch.ChapterIndex+1),
		})
	}

	return &drama, episodes, nil
}

func (p *DramaboxProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// Endpoint: /watch/player?bookId={id}&index={index}&lang=in
	idx, _ := strconv.Atoi(epIndex)

	urlPlay := fmt.Sprintf("%s/watch/player?bookId=%s&index=%d&lang=in", DramaboxAPI, id, idx)
	body, err := p.fetch(urlPlay)
	if err != nil {
		return nil, err
	}

	var resp dbResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("failed to get stream")
	}

	var playData dbPlayerData
	if err := json.Unmarshal(resp.Data, &playData); err != nil {
		return nil, err
	}

	if playData.VideoURL == "" {
		return nil, fmt.Errorf("no video url found")
	}

	return &models.StreamData{
		BookID: "dramabox:" + id,
		Chapter: models.ChapterData{
			Index: idx,
			Video: models.VideoData{
				Mp4: playData.VideoURL,
			},
		},
	}, nil
}
