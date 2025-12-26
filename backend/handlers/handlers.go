package handlers

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const BaseAPI = "https://dramabox-api-rho.vercel.app/api"

// --- External API Models (Private) ---

type ExtResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"` // Delay parsing
}

type ExtHomeData struct {
	Book []ExtBook `json:"book"`
}

type ExtSearchData struct {
	Book []ExtBookSearch `json:"book"`
}

type ExtBook struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Cover        string   `json:"cover"`
	Introduction string   `json:"introduction"`
	ChapterCount int      `json:"chapterCount"`
	Tags         []ExtTag `json:"tags"`
}

type ExtBookSearch struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Cover        string   `json:"cover"`
	Introduction string   `json:"introduction"`
	Tags         []string `json:"tags"` // Search API returns string array
}

type ExtTag struct {
	TagName string `json:"tagName"`
}

type ExtDetailData struct {
	Drama    ExtDramaDetail `json:"drama"`
	Chapters []ExtChapter   `json:"chapters"`
}

type ExtDramaDetail struct {
	BookID       string        `json:"bookId"`
	BookName     string        `json:"bookName"`
	Cover        string        `json:"cover"`
	Introduction string        `json:"introduction"`
	ChapterCount int           `json:"chapterCount"`
	Tags         []interface{} `json:"tags"` // Could be mixed, handle carefully
}

type ExtChapter struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
}

type ExtStreamResponse struct {
	Data struct {
		Chapter struct {
			ID    string `json:"id"`
			Index int    `json:"index"`
			Video struct {
				Mp4  string `json:"mp4"`
				M3u8 string `json:"m3u8"`
			} `json:"video"`
			Duration int `json:"duration"`
		} `json:"chapter"`
	} `json:"data"`
}

type ExtBookCategory struct {
	BookID       string   `json:"bookId"`
	BookName     string   `json:"bookName"`
	Cover        string   `json:"cover"`
	Introduction string   `json:"introduction"`
	ChapterCount int      `json:"chapterCount"`
	Tags         []string `json:"tags"`
}

type ExtCategoryData struct {
	BookList    []ExtBookCategory `json:"bookList"`
	CurrentPage int               `json:"currentPage"`
	Total       int64             `json:"total"`
}

type ExtCategoryResponse struct {
	Success bool            `json:"success"`
	Data    ExtCategoryData `json:"data"`
}

// --- Utils ---

func fetchExternal(url string) ([]byte, error) {
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

// --- Handlers ---

func SeedData(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "success", "message": "Using External API"})
}

func GetTrending(c *fiber.Ctx) error {
	// Proxy /api/home
	body, err := fetchExternal(BaseAPI + "/home")
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch from source"})
	}

	var raw ExtResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	var homeData ExtHomeData
	json.Unmarshal(raw.Data, &homeData)

	var dramas []models.Drama
	for _, b := range homeData.Book {
		// Map Tags
		var tags []string
		for _, t := range b.Tags {
			tags = append(tags, t.TagName)
		}

		dramas = append(dramas, models.Drama{
			BookID:       b.ID,
			Judul:        b.Name,
			Cover:        b.Cover,
			Deskripsi:    b.Introduction,
			TotalEpisode: fmt.Sprintf("%d", b.ChapterCount),
			Genre:        strings.Join(tags, ", "),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "trending",
		"data":   dramas,
	})
}

// GetLatest now uses Category 0 ("All") to support pagination and browsing
func GetLatest(c *fiber.Ctx) error {
	page := c.Query("page", "1")
	// genre := c.Query("genre") // TODO: Map genre string to ID if needed

	// Default to Category 0 (All)
	catID := "0"
	url := fmt.Sprintf("%s/category/%s?page=%s", BaseAPI, catID, page)

	body, err := fetchExternal(url)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch category data"})
	}

	var raw ExtCategoryResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		fmt.Println("JSON Error Late:", err) // Debug print
		return c.Status(500).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	var dramas []models.Drama
	for _, b := range raw.Data.BookList {
		dramas = append(dramas, models.Drama{
			BookID:       b.BookID,
			Judul:        b.BookName,
			Cover:        b.Cover,
			Deskripsi:    b.Introduction,
			TotalEpisode: fmt.Sprintf("%d", b.ChapterCount),
			Genre:        strings.Join(b.Tags, ", "),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "latest",
		"page":   raw.Data.CurrentPage,
		"total":  raw.Data.Total,
		"data":   dramas,
	})
}

func GetHero(c *fiber.Ctx) error {
	// Reuse Trending but take top 5
	// Check cache or fetch
	body, err := fetchExternal(BaseAPI + "/home")
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch"})
	}

	var raw ExtResponse
	json.Unmarshal(body, &raw)
	var homeData ExtHomeData
	json.Unmarshal(raw.Data, &homeData)

	var dramas []models.Drama
	limit := 5
	if len(homeData.Book) < 5 {
		limit = len(homeData.Book)
	}

	for i := 0; i < limit; i++ {
		b := homeData.Book[i]
		dramas = append(dramas, models.Drama{
			BookID:     b.ID,
			Judul:      b.Name,
			Cover:      b.Cover,
			Deskripsi:  b.Introduction,
			IsFeatured: true, // Force true for hero
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "hero",
		"data":   dramas,
	})
}

func GetSearch(c *fiber.Ctx) error {
	q := c.Query("q", c.Query("query"))
	if q == "" {
		return c.JSON(fiber.Map{"status": "success", "data": []models.Drama{}})
	}

	url := fmt.Sprintf("%s/search?keyword=%s", BaseAPI, q) // API uses 'keyword'
	body, err := fetchExternal(url)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Search failed"})
	}

	var raw ExtResponse
	json.Unmarshal(body, &raw)
	var searchData ExtSearchData
	json.Unmarshal(raw.Data, &searchData)

	var dramas []models.Drama
	for _, b := range searchData.Book {
		dramas = append(dramas, models.Drama{
			BookID:    b.ID,
			Judul:     b.Name,
			Cover:     b.Cover,
			Deskripsi: b.Introduction,
			Genre:     strings.Join(b.Tags, ", "), // Tags are strings here
		})
	}

	return c.JSON(fiber.Map{
		"status":        "success",
		"query":         q,
		"total_results": len(dramas),
		"data":          dramas,
	})
}

func GetDetail(c *fiber.Ctx) error {
	bookId := c.Query("bookId")
	url := fmt.Sprintf("%s/detail/%s/v2", BaseAPI, bookId)
	body, err := fetchExternal(url)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Drama not found"})
	}

	var raw ExtResponse
	json.Unmarshal(body, &raw)
	var detailData ExtDetailData
	json.Unmarshal(raw.Data, &detailData)

	// Map Episodes
	var episodes []models.Episode
	for _, ch := range detailData.Chapters {
		episodes = append(episodes, models.Episode{
			BookID:       detailData.Drama.BookID,
			EpisodeIndex: ch.Index + 1, // Convert 0-based to 1-based for frontend
			EpisodeLabel: fmt.Sprintf("Episode %d", ch.Index+1),
		})
	}

	return c.JSON(models.DetailResponse{
		Status:                "success",
		BookID:                detailData.Drama.BookID,
		Judul:                 detailData.Drama.BookName,
		Deskripsi:             detailData.Drama.Introduction,
		Cover:                 detailData.Drama.Cover,
		TotalEpisode:          fmt.Sprintf("%d", detailData.Drama.ChapterCount),
		Episodes:              episodes,
		JumlahEpisodeTersedia: len(episodes),
	})
}

func GetStream(c *fiber.Ctx) error {
	bookId := c.Query("bookId")
	idx := c.Query("index", "1") // 1-based

	// API expects 'episode' which seems to be 1-based (from our test)
	url := fmt.Sprintf("%s/stream?bookId=%s&episode=%s", BaseAPI, bookId, idx)
	body, err := fetchExternal(url)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Stream unavailable"})
	}

	var streamResp ExtStreamResponse
	if err := json.Unmarshal(body, &streamResp); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Invalid Stream JSON"})
	}

	return c.JSON(models.StreamResponse{
		Status: "success",
		Data: models.StreamData{
			BookID: bookId,
			Chapter: models.ChapterData{
				Index:    streamResp.Data.Chapter.Index, // Should match
				Duration: streamResp.Data.Chapter.Duration,
				Video: models.VideoData{
					Mp4:  streamResp.Data.Chapter.Video.Mp4,
					M3u8: streamResp.Data.Chapter.Video.M3u8,
				},
			},
		},
	})
}

func GetRandom(c *fiber.Ctx) error {
	// Fallback to Trending for now
	return GetTrending(c)
}

func GetCategories(c *fiber.Ctx) error {
	body, err := fetchExternal(BaseAPI + "/categories")
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch categories"})
	}
	// Return raw proxy
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func GetSitemapData(c *fiber.Ctx) error {
	// Unimplemented for external API yet
	return c.JSON(fiber.Map{"status": "success", "data": []string{}})
}

// --- Admin Handlers ---
// These are defined in admin.go, settings.go, upload.go, etc.
// Leaving them out of here to avoid redeclaration errors.

// --- Auth Handlers (Preserve) ---
// Note: You must ensure auth.go/users.go are still valid.
// If they are in separate files, they are fine.
// But if they were in handlers.go, I need to keep them.
// "handlers.go" contained: GetTrending, GetLatest, GetSearch, GetDetail, Stream, Random, Hero, Sitemap.
// Admin handlers were here too.
// Auth handlers (LocalLogin, VerifyGoogleToken) are in auth.go (checked file list step 1130).
// User handlers (UpdateUserProfile) are in users.go.
// So I only replaced the Content Handlers. Admin Handlers were in handlers.go too, so I stubbed them.
