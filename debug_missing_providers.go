package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	client := http.Client{Timeout: 5 * time.Second}

	patterns := []string{
		// DramaWave
		"https://dramabos.asia/api/dramawave/api/v1/home?lang=id",
		"https://dramabos.asia/api/dramawave/api/v1/home?page=1",
		"https://dramabos.asia/api/dramawave/api/home",
		"https://dramabos.asia/api/dramawave/home",
		"https://dramabos.asia/api/dramawave/v1/home",

		// FreeShort
		"https://dramabos.asia/api/freeshort/api/v1/home?lang=id",
		"https://dramabos.asia/api/freeshort/api/home",
		"https://dramabos.asia/api/freeshort/home",
		"https://dramabos.asia/api/freeshort/v1/home",

		// NetShort (Bonus check)
		"https://dramabos.asia/api/netshort/api/v1/home",
		"https://dramabos.asia/api/netshort/home",
	}

	fmt.Println("Checking endpoints...")
	for _, url := range patterns {
		check(client, url)
	}
}

func check(client http.Client, url string) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://dramabos.asia/")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERR] %s : %v\n", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Printf("[200 OK] %s\n", url)
	} else {
		fmt.Printf("[%d] %s\n", resp.StatusCode, url)
	}
}
