package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	client := http.Client{Timeout: 10 * time.Second}
	url := "https://dramabos.asia/api/dramawave/api/v1/feed/popular?lang=id"

	fmt.Printf("Fetching: %s\n", url)
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

	s := string(body)
	if len(s) > 1000 {
		fmt.Printf("Response (first 1000): %s\n", s[:1000])
	} else {
		fmt.Printf("Response: %s\n", s)
	}
}
