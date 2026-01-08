package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	urls := []string{
		"https://api.sansekai.my.id/api/dramabox/vip",
		"https://api.sansekai.my.id/api/dramabox/foryou",
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, url := range urls {
		fmt.Println("Fetching:", url)
		req, _ := http.NewRequest("GET", url, nil)

		// Exact headers from dramabox.go
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Referer", "https://dramabox.com/")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		defer resp.Body.Close()

		fmt.Println("Status:", resp.Status)
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Body:", string(body))
		fmt.Println("---------------------------------------------------")
	}
}
