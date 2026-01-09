package adapter

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type FreeShortProvider struct{}

const FreeShortAPI = "https://sapimu.au/freeshort/api/v1"

func NewFreeShortProvider() *FreeShortProvider {
	return &FreeShortProvider{}
}

func (p *FreeShortProvider) GetID() string {
	return "freeshort"
}

func (p *FreeShortProvider) IsCompatibleID(id string) bool {
	return true
}

func (p *FreeShortProvider) fetch(targetURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+SapimuToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (p *FreeShortProvider) proxyImage(originalURL string) string {
	if originalURL == "" {
		return ""
	}
	// Use wsrv.nl for simple proxy/resize
	return "https://wsrv.nl/?url=" + url.QueryEscape(originalURL) + "&output=jpg"
}

// Reuse structs from Starshort if structure is identical (likely)
// But Go doesn't like cross-file private structs easily unless exported.
// I will copy private structs to be safe and independent.

type fsResponse struct {
	Items []fsItem `json:"items"`
}
type fsItem struct {
	Key   string   `json:"key"`
	Title string   `json:"title"`
	Cover string   `json:"cover"`
	Tags  []string `json:"tag"`
}

// fsDetail and fsEpisode reuse from before or generic

func (p *FreeShortProvider) GetTrending() ([]models.Drama, error) {
	// Use /foryou for Trending
	body, err := p.fetch(FreeShortAPI + "/foryou?lang=id-ID")
	if err != nil {
		return nil, err
	}

	var raw fsResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var dramas []models.Drama
	for _, d := range raw.Items {
		dramas = append(dramas, models.Drama{
			BookID: "freeshort:" + d.Key,
			Judul:  d.Title,
			Cover:  p.proxyImage(d.Cover),
			Genre:  "", // d.Tags is []string
		})
	}
	return dramas, nil
}

func (p *FreeShortProvider) GetLatest(page int) ([]models.Drama, error) {
	// Use /foryou for Latest too (as we don't have explicit /new)
	return p.GetTrending()
}

func (p *FreeShortProvider) Search(query string) ([]models.Drama, error) {
	return []models.Drama{}, nil
}

func (p *FreeShortProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	return nil, nil, fmt.Errorf("freeshort api unavailable (500)")
}

func (p *FreeShortProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	return nil, fmt.Errorf("freeshort api unavailable (500)")
}
