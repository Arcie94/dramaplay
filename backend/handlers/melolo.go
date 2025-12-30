package handlers

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const MeloloAPI = "https://api.sansekai.my.id/api/melolo"

// --- Melolo External Models ---

type MeloloBook struct {
	BookID             string   `json:"book_id"`
	BookName           string   `json:"book_name"`
	ThumbURL           string   `json:"thumb_url"`
	Abstract           string   `json:"abstract"`
	Author             string   `json:"author"`
	ShowCreationStatus string   `json:"show_creation_status"`
	SerialCount        string   `json:"serial_count"`
	StatInfos          []string `json:"stat_infos"`
}

type MeloloListResponse struct {
	Algo  int          `json:"algo"`
	Books []MeloloBook `json:"books"`
}

type MeloloSearchResponse struct {
	Code int `json:"code"`
	Data struct {
		SearchData []struct {
			Books []MeloloBook `json:"books"`
			Name  string       `json:"name"`
		} `json:"search_data"`
	} `json:"data"`
}

// --- Handlers ---

func FetchMelolo(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func ProxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	// Use wsrv.nl to proxy and convert to jpg
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

func GetMeloloLatest(c *fiber.Ctx) error {
	body, err := FetchMelolo(MeloloAPI + "/latest")
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Melolo API unreachable"})
	}

	var raw MeloloListResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Invalid JSON from Melolo"})
	}

	var dramas []models.Drama
	for _, b := range raw.Books {
		dramas = append(dramas, models.Drama{
			BookID:       b.BookID,
			Judul:        b.BookName,
			Cover:        ProxyImage(b.ThumbURL),
			Deskripsi:    b.Abstract,
			TotalEpisode: b.SerialCount,
			Genre:        strings.Join(b.StatInfos, ", "),
			IsFeatured:   false,
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"source": "melolo",
		"type":   "latest",
		"data":   dramas,
	})
}

func GetMeloloTrending(c *fiber.Ctx) error {
	body, err := FetchMelolo(MeloloAPI + "/trending")
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Melolo API unreachable"})
	}

	var raw MeloloListResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Invalid JSON from Melolo"})
	}

	var dramas []models.Drama
	for _, b := range raw.Books {
		dramas = append(dramas, models.Drama{
			BookID:       b.BookID,
			Judul:        b.BookName,
			Cover:        ProxyImage(b.ThumbURL),
			Deskripsi:    b.Abstract,
			TotalEpisode: b.SerialCount,
			Genre:        strings.Join(b.StatInfos, ", "),
			IsFeatured:   false,
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"source": "melolo",
		"type":   "trending",
		"data":   dramas,
	})
}

func GetMeloloSearch(c *fiber.Ctx) error {
	query := c.Query("q", "")
	if query == "" {
		return c.JSON(fiber.Map{"status": "success", "data": []interface{}{}})
	}

	url := fmt.Sprintf("%s/search?query=%s", MeloloAPI, query)
	body, err := FetchMelolo(url)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "Melolo API unreachable"})
	}

	var raw MeloloSearchResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Invalid Search JSON"})
	}

	var dramas []models.Drama
	// Flatten nested search data (Melolo groups by category in search results)
	for _, group := range raw.Data.SearchData {
		for _, b := range group.Books {
			dramas = append(dramas, models.Drama{
				BookID:       b.BookID,
				Judul:        b.BookName,
				Cover:        ProxyImage(b.ThumbURL),
				Deskripsi:    b.Abstract,
				TotalEpisode: b.SerialCount,
				Genre:        strings.Join(b.StatInfos, ", "),
				IsFeatured:   false,
			})
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"source": "melolo",
		"query":  query,
		"data":   dramas,
	})
}
