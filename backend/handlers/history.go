package handlers

import (
	"dramabang/database"
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type HistoryInput struct {
	UserID     uint   `json:"userId"`
	BookID     string `json:"bookId"`
	EpisodeIdx int    `json:"episodeIdx"`
}

func SaveHistory(c *fiber.Ctx) error {
	var input HistoryInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Invalid input"})
	}

	var history models.UserHistory
	// Check if exists
	result := database.DB.Where("user_id = ? AND book_id = ?", input.UserID, input.BookID).First(&history)

	if result.Error != nil {
		// Create new
		history = models.UserHistory{
			UserID:     input.UserID,
			BookID:     input.BookID,
			EpisodeIdx: input.EpisodeIdx,
		}
		if err := database.DB.Create(&history).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to create history"})
		}
	} else {
		// Update
		history.EpisodeIdx = input.EpisodeIdx
		if err := database.DB.Save(&history).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to update history"})
		}
	}

	return c.JSON(fiber.Map{"status": "success"})
}

func GetHistory(c *fiber.Ctx) error {
	userId := c.Query("userId")
	if userId == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Missing userId"})
	}

	var histories []models.UserHistory
	// Preload Drama to get Title/Cover
	err := database.DB.Preload("Drama").Where("user_id = ?", userId).Order("updated_at desc").Find(&histories).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Database error"})
	}

	// Lazy Ingest: Ensure Drama details exist. Only needed if Preload found nothing (BookID exists but no Drama row)
	// Actually Preload won't fill .Drama if it's missing.
	// Check if any history has empty Drama data
	for i, h := range histories {
		if h.Drama.BookID == "" {
			// Drama missing in local DB (likely from Trending proxy)
			// Fetch from external API and save
			// Fetch from external API and save (Blocking for first load UX)
			url := fmt.Sprintf("%s/detail/%s/v2", BaseAPI, h.BookID)
			resp, err := http.Get(url)
			if err == nil && resp.StatusCode == 200 {
				body, _ := io.ReadAll(resp.Body)

				var raw ExtResponse
				if json.Unmarshal(body, &raw) == nil {
					var detailData ExtDetailData
					if json.Unmarshal(raw.Data, &detailData) == nil && detailData.Drama.BookID != "" {
						// Map Tags
						var tags []string
						for _, t := range detailData.Drama.Tags {
							if s, ok := t.(string); ok {
								tags = append(tags, s)
							} else if m, ok := t.(map[string]interface{}); ok {
								if tagName, ok := m["tagName"].(string); ok {
									tags = append(tags, tagName)
								}
							}
						}

						newDrama := models.Drama{
							BookID:       detailData.Drama.BookID,
							Judul:        detailData.Drama.BookName,
							Cover:        detailData.Drama.Cover,
							Deskripsi:    detailData.Drama.Introduction,
							TotalEpisode: fmt.Sprintf("%d", detailData.Drama.ChapterCount),
							Genre:        strings.Join(tags, ", "),
						}

						database.DB.Save(&newDrama)
						histories[i].Drama = newDrama
					}
				}
				resp.Body.Close()
			}
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   histories,
	})
}
