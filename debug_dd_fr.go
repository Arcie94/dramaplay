package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	client := http.Client{Timeout: 15 * time.Second}

	// 1. Check FlickReels again (maybe different endpoint?)
	// User said: https://dramabos.asia/api/flick/home?page=1&page_size=10&lang=6
	checkEndpoint("FlickReels", "https://dramabos.asia/api/flick/home?page=1&page_size=10&lang=6", client)

	// 2. Check DramaDash
	// User said: https://dramabos.asia/api/dramadash/api/tabs/1
	checkEndpoint("DramaDash Tabs", "https://dramabos.asia/api/dramadash/api/tabs/1", client)

	// Check DramaDash Home
	checkEndpoint("DramaDash Home", "https://dramabos.asia/api/dramadash/api/home", client)
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
