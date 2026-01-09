package main

import (
	"fmt"
	"io"
	"net/http"
)

const Token = "0ebd6cfdd8054d2a90aa2851532645211aeaf189fa1aed62c53e5fd735af8649"

func fetch(name, url string) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+Token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[%s] Error: %v\n", name, err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// Print first 500 chars to see structure
	limit := 1000
	if len(body) < limit {
		limit = len(body)
	}
	fmt.Printf("[%s] Endpoint: %s\nStatus: %d\nBody: %s\n\n", name, url, resp.StatusCode, string(body[:limit]))
}

func main() {
	// FreeShort
	fetch("FreeShort", "https://sapimu.au/freeshort/api/v1/foryou?lang=id-ID")

	// ShortMax
	fetch("ShortMax", "https://sapimu.au/shortmax/api/v1/home?lang=id")

	// DramaDash
	fetch("DramaDash", "https://sapimu.au/dramadash/home?lang=in")

	// HiShort
	fetch("HiShort", "https://sapimu.au/hishort/api/home?lang=in")

	// FlickReels (Guessing)
	fetch("FlickReels", "https://sapimu.au/flickreels/api/v1/home?lang=en") // Default to EN to test existence first

	// DramaWave (Guessing)
	fetch("DramaWave", "https://sapimu.au/dramawave/api/v1/home?lang=en")
}
