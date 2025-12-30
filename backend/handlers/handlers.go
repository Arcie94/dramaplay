package handlers

import (
	"dramabang/models"
	"dramabang/services/adapter"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

var AdapterManager *adapter.Manager

// Initialize Manager on startup
func init() {
	AdapterManager = adapter.NewManager()
}

// --- Handlers ---

func SeedData(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "success", "message": "Using Universal Adapter"})
}

func GetTrending(c *fiber.Ctx) error {
	dramas, err := AdapterManager.GetTrending()
	if err != nil {
		// If partial error, we might still have data?
		// Manager currently logs errors and returns what it has.
		// If slice is nil, return error?
		// Empty slice is valid.
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "trending",
		"data":   dramas,
	})
}

func GetLatest(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}

	dramas, err := AdapterManager.GetLatest(page)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch latest"})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "latest",
		"page":   page,
		"data":   dramas,
	})
}

func GetHero(c *fiber.Ctx) error {
	// Reuse Trending but take top 5
	dramas, err := AdapterManager.GetTrending()
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch hero"})
	}

	limit := 5
	if len(dramas) < limit {
		limit = len(dramas)
	}

	var heroDramas []models.Drama
	for i := 0; i < limit; i++ {
		d := dramas[i]
		d.IsFeatured = true
		heroDramas = append(heroDramas, d)
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "hero",
		"data":   heroDramas,
	})
}

func GetSearch(c *fiber.Ctx) error {
	q := c.Query("q", c.Query("query"))
	if q == "" {
		return c.JSON(fiber.Map{"status": "success", "data": []models.Drama{}})
	}

	dramas, err := AdapterManager.Search(q)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Search failed"})
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
	if bookId == "" {
		// Support path param if routed that way, but existing uses query
		bookId = c.Params("book_id") // Fallback
	}

	drama, episodes, err := AdapterManager.GetDetail(bookId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Drama not found or provider error: " + err.Error()})
	}

	// Map to response format expected by frontend
	return c.JSON(models.DetailResponse{
		Status:                "success",
		BookID:                drama.BookID,
		Judul:                 drama.Judul,
		Deskripsi:             drama.Deskripsi,
		Cover:                 drama.Cover,
		TotalEpisode:          drama.TotalEpisode,
		Episodes:              episodes,
		JumlahEpisodeTersedia: len(episodes),
	})
}

func GetStream(c *fiber.Ctx) error {
	bookId := c.Query("bookId")
	idxStr := c.Query("index", "1") // Assuming 1-based index from frontend

	// Adapter expects string index because providers might handle different indexing
	// But our interface uses string index.
	streamData, err := AdapterManager.GetStream(bookId, idxStr)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Stream not found: " + err.Error()})
	}

	return c.JSON(models.StreamResponse{
		Status: "success",
		Data:   *streamData,
	})
}

func GetRandom(c *fiber.Ctx) error {
	return GetTrending(c)
}

func GetCategories(c *fiber.Ctx) error {
	// Unimplemented in Adapter.
	// Return empty or fetch from just Dramabox?
	// For now, return empty to avoid errors
	return c.JSON(fiber.Map{"status": "success", "data": []interface{}{}})
}

func GetSitemapData(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "success", "data": []string{}})
}
