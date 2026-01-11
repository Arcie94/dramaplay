package handlers

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ProxyStream handles HLS proxying to bypass CORS and inject tokens/rewrite paths
func ProxyStream(c *fiber.Ctx) error {
	targetURL := c.Query("url")
	if targetURL == "" {
		return c.Status(400).SendString("Missing url param")
	}

	// Fetch the target resource
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return c.Status(500).SendString("Invalid URL")
	}

	// Forward User-Agent for compatibility
	req.Header.Set("User-Agent", c.Get("User-Agent"))

	// Create client (disable compression to easily rewrite text)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(502).SendString("Failed to fetch upstream: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Status(resp.StatusCode).SendString("Upstream error")
	}

	// Set CORS headers
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Content-Type", resp.Header.Get("Content-Type"))

	// Determine if it's an M3U8 Playlist that needs rewriting
	contentType := resp.Header.Get("Content-Type")
	isM3U8 := strings.Contains(contentType, "mpegurl") || strings.Contains(contentType, "m3u8") || strings.HasSuffix(targetURL, ".m3u8")

	if isM3U8 {
		// Use baseUrl to resolve relative paths
		u, _ := url.Parse(targetURL)

		scanner := bufio.NewScanner(resp.Body)
		var sb strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			if strings.HasPrefix(trimmed, "#") {
				sb.WriteString(line + "\n")
				continue
			}

			if trimmed == "" {
				continue
			}

			// It's a URI (Segment or Key or Sub-playlist)
			// Resolve absolute URL
			var absoluteURI string
			if strings.HasPrefix(trimmed, "http") {
				absoluteURI = trimmed
			} else {
				// Resolve relative
				rel, _ := url.Parse(trimmed)
				absoluteURI = u.ResolveReference(rel).String()
			}

			// Rewrite to Proxy URL
			// We point back to /api/proxy?url=
			proxyURL := fmt.Sprintf("/api/proxy?url=%s", url.QueryEscape(absoluteURI))
			sb.WriteString(proxyURL + "\n")
		}

		return c.SendString(sb.String())
	}

	// For segments/keys (binary), just stream copy
	return c.Status(resp.StatusCode).SendStream(resp.Body)
}
