package adapter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"dramabang/models"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type MovieProvider struct {
	client  *http.Client
	baseURL string
	ajaxURL string
	keys    []string
}

func NewMovieProvider() *MovieProvider {
	return &MovieProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://tv12.idlixku.com",
		ajaxURL: "https://tv12.idlixku.com/wp-admin/admin-ajax.php",
		keys: []string{
			"459283", "idlix", "id123", "root", "dooplay",
			"tv12.idlixku.com", "https://tv12.idlixku.com/", "Dooplay", "admin",
		},
	}
}

func (p *MovieProvider) GetID() string {
	return "movie"
}

func (p *MovieProvider) IsCompatibleID(id string) bool {
	return strings.HasPrefix(id, "movie:")
}

// --- HTTP Helper with Headers ---
func (p *MovieProvider) request(method, targetURL string, data url.Values) (*http.Response, error) {
	var req *http.Request
	var err error

	if method == "POST" {
		req, err = http.NewRequest("POST", targetURL, strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest("GET", targetURL, nil)
		if err != nil {
			return nil, err
		}
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", p.baseURL)
	req.Header.Set("Origin", p.baseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	return p.client.Do(req)
}

func (p *MovieProvider) fetchDOM(targetURL string) (*goquery.Document, error) {
	resp, err := p.request("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// --- Implementation ---

func (p *MovieProvider) GetTrending() ([]models.Drama, error) {
	doc, err := p.fetchDOM(p.baseURL)
	if err != nil {
		return nil, err
	}

	var movies []models.Drama
	doc.Find(".item.movies, .item.tvshows").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".data h3 a").Text())
		link, _ := s.Find(".data h3 a").Attr("href")
		poster, _ := s.Find(".poster img").Attr("src")

		// Extract ID from link (last segment usually) or just use full Link as ID (encoded)
		// Better to use full link or slug. Let's use link but strip base.
		// Example: https://tv12.idlixku.com/movie-title/ -> movie-title
		id := strings.TrimPrefix(link, p.baseURL+"/")
		id = strings.Trim(id, "/")

		if title != "" && id != "" {
			movies = append(movies, models.Drama{
				BookID:    "movie:" + id,
				Judul:     title,
				Cover:     poster,
				Genre:     "Movie", // Default
				Deskripsi: "",
			})
		}
	})
	return movies, nil
}

func (p *MovieProvider) Search(query string) ([]models.Drama, error) {
	searchURL := fmt.Sprintf("%s/?s=%s", p.baseURL, url.QueryEscape(query))
	doc, err := p.fetchDOM(searchURL)
	if err != nil {
		return nil, err
	}

	var movies []models.Drama
	doc.Find(".search-item, .result-item").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".title a").Text())
		link, _ := s.Find(".title a").Attr("href")
		poster, _ := s.Find(".thumbnail img").Attr("src")

		if title == "" {
			// Fallback selectors
			title = strings.TrimSpace(s.Find("h3 a").Text())
			link, _ = s.Find("h3 a").Attr("href")
			poster, _ = s.Find("img").Attr("src")
		}

		id := strings.TrimPrefix(link, p.baseURL+"/")
		id = strings.Trim(id, "/")

		if title != "" && id != "" {
			movies = append(movies, models.Drama{
				BookID: "movie:" + id,
				Judul:  title,
				Cover:  poster,
				Genre:  "Movie",
			})
		}
	})
	return movies, nil
}

func (p *MovieProvider) GetLatest(page int) ([]models.Drama, error) {
	// Same as Trending but with page
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in MovieProvider.GetLatest:", r)
		}
	}()

	targetURL := p.baseURL
	if page > 1 {
		targetURL = fmt.Sprintf("%s/page/%d/", p.baseURL, page)
	}

	doc, err := p.fetchDOM(targetURL)
	if err != nil {
		return nil, err
	}

	var movies []models.Drama
	doc.Find(".item.movies").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".data h3 a").Text())
		link, _ := s.Find(".data h3 a").Attr("href")
		poster, _ := s.Find(".poster img").Attr("src")

		id := strings.TrimPrefix(link, p.baseURL+"/")
		id = strings.Trim(id, "/")

		if title != "" && id != "" {
			movies = append(movies, models.Drama{
				BookID: "movie:" + id,
				Judul:  title,
				Cover:  poster,
				Genre:  "Movie",
			})
		}
	})
	return movies, nil
}

func (p *MovieProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// Reconstruct URL
	targetURL := fmt.Sprintf("%s/%s/", p.baseURL, id)
	doc, err := p.fetchDOM(targetURL)
	if err != nil {
		return nil, nil, err
	}

	title := strings.TrimSpace(doc.Find(".data h1").Text())
	synopsis := strings.TrimSpace(doc.Find(".wp-content p").First().Text())
	cover, _ := doc.Find(".poster img").Attr("src")

	// Episodes / Players
	// In IDLIX movie, it's usually just "Movie" (1 episode) but with multiple servers.
	// We treat servers as "Episodes" or just 1 Episode?
	// The scraper logic `get_movie_detail` finds `#playeroptionsul li`.
	// For now, let's just map 1 Episode calling it "Movie" unless it is a TV Show.

	var episodes []models.Episode

	// Check if it has seasons/episodes (TV Show)
	if doc.Find("#seasons").Length() > 0 {
		doc.Find(".se-c").Each(func(i int, s *goquery.Selection) {
			_ = s.Find(".se-q").Text() // Episode Number (Unused for now)
			_, _ = s.Find(".se-t .title a").Attr("href")
			// We need to store the link or ID of the episode page.
			// Currently `GetStream` takes (id, epIndex).
			// If it's a TV show, the ID for stream needs to be the Episode Page ID.
			// This complicates the "movie:" prefix logic if we switch pages.
			// Simplified: For now support Movies only as requested?
			// User said "Scraping API IDLIX untuk menambah library... movie.go".
			// Let's assume Movies first.
		})
	}

	// For Movies, we look for #playeroptionsul to verify it exists
	if doc.Find("#playeroptionsul li").Length() > 0 {
		episodes = append(episodes, models.Episode{
			BookID:       "movie:" + id,
			EpisodeIndex: 0,
			EpisodeLabel: "Full Movie",
		})
	}

	drama := &models.Drama{
		BookID:       "movie:" + id,
		Judul:        title,
		Deskripsi:    synopsis,
		Cover:        cover,
		TotalEpisode: "1",
		Genre:        "Movie",
	}

	return drama, episodes, nil
}

func (p *MovieProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	// 1. Fetch Detail Page again to get Player Options (PostID, Nume, Type)
	targetURL := fmt.Sprintf("%s/%s/", p.baseURL, id)
	doc, err := p.fetchDOM(targetURL)
	if err != nil {
		return nil, err
	}

	// Dynamic Key Update (Nonce)
	// var nonce string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "nonce") {
			re := regexp.MustCompile(`"nonce":"(\w+)"`)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				p.keys = append([]string{matches[1]}, p.keys...) // Prepend
			}
		}
	})

	// Find first player option (usually "Server 1" or similar)
	// Or try to find "HLS" or "Grive"
	var postID, nume, vType string

	// Priority: Google Drive / VIP > others
	selection := doc.Find("#playeroptionsul li").First()
	postID, _ = selection.Attr("data-post")
	nume, _ = selection.Attr("data-nume")
	vType, _ = selection.Attr("data-type")

	if postID == "" {
		return nil, fmt.Errorf("no player found")
	}

	// 2. Hit Ajax
	data := url.Values{}
	data.Set("action", "doo_player_ajax")
	data.Set("post", postID)
	data.Set("nume", nume)
	data.Set("type", vType)

	resp, err := p.request("POST", p.ajaxURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	respText := string(bodyBytes)

	// 3. Decrypt if needed
	var finalURL string
	if strings.Contains(respText, "\"ct\":") {
		// Encrypted
		var encryptedJSON struct {
			CT string `json:"ct"`
			IV string `json:"iv"`
			S  string `json:"s"`
		}
		if err := json.Unmarshal(bodyBytes, &encryptedJSON); err == nil {
			// Try Keys
			for _, key := range p.keys {
				decrypted, err := DecryptAES(encryptedJSON.CT, encryptedJSON.IV, encryptedJSON.S, key)
				if err == nil && decrypted != "" {
					// Clean result
					decrypted = strings.Trim(decrypted, "\"")
					decrypted = strings.ReplaceAll(decrypted, "\\/", "/")

					// It might be JSON inside
					if strings.HasPrefix(decrypted, "{") {
						var inner struct {
							EmbedURL string `json:"embed_url"`
							Type     string `json:"type"`
						}
						json.Unmarshal([]byte(decrypted), &inner)
						finalURL = inner.EmbedURL
					} else if strings.Contains(decrypted, "<iframe") {
						// Parse HTML
						dDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(decrypted))
						finalURL, _ = dDoc.Find("iframe").Attr("src")
					} else {
						finalURL = decrypted
					}
					break
				}
			}
		}
	} else {
		// Plain JSON or HTML
		if strings.HasPrefix(respText, "{") {
			var inner struct {
				EmbedURL string `json:"embed_url"`
			}
			json.Unmarshal(bodyBytes, &inner)
			finalURL = inner.EmbedURL
		}
	}

	if finalURL == "" {
		// Fallback: Check if iframe in raw text
		if strings.Contains(respText, "<iframe") {
			dDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(respText))
			finalURL, _ = dDoc.Find("iframe").Attr("src")
		}
	}

	if finalURL == "" {
		return nil, fmt.Errorf("failed to extract stream url")
	}

	return &models.StreamData{
		BookID: "movie:" + id,
		Chapter: models.ChapterData{
			Index: 0,
			Video: models.VideoData{
				Mp4: finalURL, // Usually iframe URL, Frontend VideoPlayer needs to handle it (iframe support)
			},
		},
	}, nil
}

// --- Crypto Utils ---

func DecryptAES(ctB64, ivHex, saltHex, password string) (string, error) {
	ct, _ := base64.StdEncoding.DecodeString(ctB64)
	iv, _ := hex.DecodeString(ivHex)
	salt, _ := hex.DecodeString(saltHex)
	pass := []byte(password)

	// EVP_BytesToKey (MD5)
	key, _ := evpBytesToKey(pass, salt, 32, 16) // AES-256

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(ct)%aes.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ct, ct)

	// Unpad (PKCS7)
	padding := int(ct[len(ct)-1])
	if padding > len(ct) || padding == 0 {
		return "", fmt.Errorf("invalid padding")
	}
	return string(ct[:len(ct)-padding]), nil
}

func evpBytesToKey(password, salt []byte, keyLen, ivLen int) ([]byte, []byte) {
	var m []byte
	var dt []byte
	var key, iv []byte

	for len(dt) < keyLen+ivLen {
		h := md5.New()
		if len(m) > 0 {
			h.Write(m)
		}
		h.Write(password)
		h.Write(salt)
		m = h.Sum(nil)
		dt = append(dt, m...)
	}

	key = dt[:keyLen]
	iv = dt[keyLen : keyLen+ivLen]
	return key, iv
}
