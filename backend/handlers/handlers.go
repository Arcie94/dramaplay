package handlers

import (
	"dramabang/database"
	"dramabang/models"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// Helper to proxy requests
func proxyRequest(c *fiber.Ctx, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL:", url, err)
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to connect to source"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("Error status from URL:", url, resp.StatusCode)
		return c.Status(resp.StatusCode).JSON(fiber.Map{"status": "error", "message": "Source unavailable"})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body:", err)
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to read data"})
	}

	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func SeedData(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "success", "message": "Seeding deprecated. Using external API."})
}

func GetTrending(c *fiber.Ctx) error {
	return proxyRequest(c, "https://dramabox-asia.vercel.app/api/trending")
}

func GetLatest(c *fiber.Ctx) error {
	var dramas []models.Drama
	var total int64

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit := 12
	offset := (page - 1) * limit

	// Genre Query
	genre := c.Query("genre")
	if genre != "" {
		fmt.Printf("GetLatest Filter: Genre='%s'\n", genre)
	}

	database.DB.Model(&models.Drama{}).Count(&total)

	query := database.DB.Limit(limit).Offset(offset).Order("book_id desc")
	if genre != "" {
		query = query.Where("genre = ?", genre)
	}
	query.Find(&dramas)

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "latest",
		"page":   page,
		"total":  total,
		"data":   dramas,
	})
}

func GetSearch(c *fiber.Ctx) error {
	query := c.Query("q") // Frontend sends 'q' mostly, Astro config sends 'q' to proxy, but let's support 'q'
	if query == "" {
		query = c.Query("query")
	}

	var dramas []models.Drama
	database.DB.Where("judul LIKE ?", "%"+query+"%").Limit(100).Find(&dramas)

	return c.JSON(fiber.Map{
		"status":        "success",
		"query":         query,
		"total_results": len(dramas),
		"data":          dramas,
	})
}

func GetDetail(c *fiber.Ctx) error {
	bookId := c.Query("bookId")

	// Try to find in DB first to be fast, or fallback to proxy if we want fresh episodes?
	// For "Detail", usually we need the list of episodes.
	// Our ingest script DOES NOT ingest the episodes list for every drama (it loops 'latest' which returns dramas, but does it return episodes?).
	// The 'latest' endpoint from scraper returns Drama Basic Info.
	// To get episodes, we need to call /api/detail for EACH drama.
	// PROPOSAL: Keep GetDetail PROXIED for now to ensure we get the full episode list freshly.
	// OR: Modify ingest to fetch detail for each... that would be too slow (6000 requests).
	// DECISION: Keep GetDetail PROXIED. Local DB is for Listing/Searching.
	return proxyRequest(c, fmt.Sprintf("https://dramabox-asia.vercel.app/api/detail?bookId=%s", bookId))
}

func GetStream(c *fiber.Ctx) error {
	bookId := c.Query("bookId")
	indexStr := c.Query("index")

	if indexStr == "" {
		indexStr = "1"
	}

	// External API uses 'episode' param, we use 'index'
	apiUrl := fmt.Sprintf("https://dramabox-asia.vercel.app/api/stream?bookId=%s&episode=%s", bookId, indexStr)
	return proxyRequest(c, apiUrl)
}

func GetRandom(c *fiber.Ctx) error {
	limit := 12
	var dramas []models.Drama

	// Optional Genre Filter
	genre := c.Query("genre")

	query := database.DB.Order("RANDOM()").Limit(limit)
	if genre != "" {
		query = query.Where("genre = ?", genre)
	}

	query.Find(&dramas)

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "random",
		"data":   dramas,
	})
}

func GetHero(c *fiber.Ctx) error {
	var dramas []models.Drama
	// Find ALL featured dramas
	err := database.DB.Where("is_featured = ?", true).Find(&dramas).Error

	if err != nil || len(dramas) == 0 {
		// No featured drama found, return empty list
		return c.JSON(fiber.Map{
			"status": "success",
			"type":   "hero",
			"data":   []models.Drama{},
		})
	}

	// Lazy load descriptions if missing
	for i, d := range dramas {
		if d.Deskripsi == "" {
			// Fetch from external API to get details
			url := fmt.Sprintf("https://dramabox-asia.vercel.app/api/detail?bookId=%s", d.BookID)
			resp, err := http.Get(url)
			if err == nil && resp.StatusCode == 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				var detail models.DetailResponse
				if err := json.Unmarshal(body, &detail); err == nil && detail.Deskripsi != "" {
					dramas[i].Deskripsi = detail.Deskripsi
					// Update Database
					database.DB.Model(&d).Update("deskripsi", detail.Deskripsi)
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"type":   "hero",
		"data":   dramas,
	})
}

func GetSitemapData(c *fiber.Ctx) error {
	type SitemapItem struct {
		BookID string `json:"bookId"`
	}
	var items []SitemapItem
	// Fetch all BookIDs
	database.DB.Model(&models.Drama{}).Select("book_id").Find(&items)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   items,
	})
}
