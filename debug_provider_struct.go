package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	client := http.Client{Timeout: 10 * time.Second}

	// 1. FlickReels
	checkEndpoint("FlickReels", "https://dramabos.asia/api/flick/home?page=1&page_size=10&lang=6", client)

	// 2. HiShort
	checkEndpoint("HiShort", "https://dramabos.asia/api/hishort/api/v1/home?module=12&page=1", client)

	// 3. DramaWave (Re-check)
	checkEndpoint("DramaWave", "https://dramabos.asia/api/dramawave/api/v1/home?page=1", client)
}

func checkEndpoint(name, url string, client http.Client) {
	fmt.Printf("\n=== Checking %s: %s ===\n", name, url)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://dramabos.asia/")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		s := string(body)
		if len(s) > 1000 {
			fmt.Printf("Response (first 1000): %s\n", s[:1000])
		} else {
			fmt.Printf("Response: %s\n", s)
		}
	} else {
		fmt.Printf("Error Body: %s\n", string(body))
	}
}
