package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	url := "https://dramabos.asia/api/starshort/api/v1/home?lang=4"
	fmt.Println("Fetching:", url)

	req, _ := http.NewRequest("GET", url, nil)
	// Add browser headers
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

	// Print first 500 chars to see structure
	s := string(body)
	if len(s) > 500 {
		fmt.Println("Response:", s[:500])
	} else {
		fmt.Println("Response:", s)
	}
}
