package handlers

import (
	"dramabang/models"

	"github.com/gofiber/fiber/v2"
)

// Legacy BaseAPI constant removed as we use AdapterManager now

// --- Handlers ---

func SeedData(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "success", "message": "Using Universal Adapter Manager"})
}

func GetTrending(c *fiber.Ctx) error {
	dramas, err := AdapterManager.GetTrending()
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch trending data", "details": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "trending",
		"data":   dramas,
	})
}

func GetLatest(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)

	dramas, err := AdapterManager.GetLatest(page)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch latest data", "details": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "latest",
		"page":   page,
		"data":   dramas,
	})
}

func GetProviderLatest(c *fiber.Ctx) error {
	provider := c.Params("provider")
	page := c.QueryInt("page", 1)

	dramas, err := AdapterManager.GetLatestFromProvider(provider, page)
	if err != nil {
		// Return empty list instead of error to prevent frontend crash
		return c.JSON(fiber.Map{
			"status": "success",
			"type":   "latest_provider",
			"data":   []models.Drama{},
		})
	}

	return c.JSON(fiber.Map{
		"status":   "success",
		"type":     "latest_provider",
		"provider": provider,
		"page":     page,
		"data":     dramas,
	})
}

func GetHero(c *fiber.Ctx) error {
	// Hero uses top 5 trending dramas
	dramas, err := AdapterManager.GetTrending()
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Failed to fetch hero data"})
	}

	limit := 5
	if len(dramas) < limit {
		limit = len(dramas)
	}

	heroDramas := make([]models.Drama, limit)
	for i := 0; i < limit; i++ {
		heroDramas[i] = dramas[i]
		heroDramas[i].IsFeatured = true
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
		return c.Status(502).JSON(fiber.Map{"error": "Search failed", "details": err.Error()})
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
		return c.Status(400).JSON(fiber.Map{"error": "Missing bookId"})
	}

	drama, episodes, err := AdapterManager.GetDetail(bookId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Drama not found", "details": err.Error()})
	}

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
	idx := c.Query("index", "1") // 1-based index usually, but provider handles it

	if bookId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Missing bookId"})
	}

	data, err := AdapterManager.GetStream(bookId, idx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Stream unavailable", "details": err.Error()})
	}

	return c.JSON(models.StreamResponse{
		Status: "success",
		Data:   *data,
	})
}

func GetRandom(c *fiber.Ctx) error {
	// Fallback to Trending for now
	return GetTrending(c)
}

func GetCategories(c *fiber.Ctx) error {
	// Not implemented in generic adapter yet
	return c.JSON(fiber.Map{"status": "success", "data": []string{}})
}

func GetSitemapData(c *fiber.Ctx) error {
	// Unimplemented for external API yet
	return c.JSON(fiber.Map{"status": "success", "data": []string{}})
}
