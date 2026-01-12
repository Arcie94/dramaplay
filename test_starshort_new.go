package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	// Test Starshort endpoints
	testUrls := []string{
		"https://dramabos.asia/api/starshort/api/v1/home?lang=4",
		"https://dramabos.asia/api/starshort/api/v1/search?q=cinta&lang=4",
		"https://dramabos.asia/api/starshort/api/v1/drama/myn?lang=4", // Assuming 'myn' is an ID from home response, but let's test this raw URL first to see if it works or if I need a real ID
		// Wait, 'myn' looks like a placeholder ID in your example.
		// I'll first fetch home to get a real ID, then test detail.

		"https://dramabos.asia/api/starshort/api/v1/rank?lang=4",
	}

	fmt.Println("Step 1: Fetch Home to get real ID")
	resp, err := http.Get(testUrls[0])
	if err != nil {
		fmt.Println("Error fetching home:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Home Status: %d\n", resp.StatusCode)
	if len(body) > 500 {
		fmt.Printf("Home Response (first 500): %s...\n", string(body[:500]))
	} else {
		fmt.Printf("Home Response: %s\n", string(body))
	}

	// I need a real ID to test Detail/Stream.
	// I will just print the Home response first, then I can parse it visually or programmatically in next step if needed.
	// But let's also test 'myn' URL just in case it's a specific test ID or something.

	fmt.Println("\nStep 2: Test provided 'myn' URL")
	resp2, err := http.Get("https://dramabos.asia/api/starshort/api/v1/drama/myn?lang=4")
	if err == nil {
		defer resp2.Body.Close()
		body2, _ := io.ReadAll(resp2.Body)
		fmt.Printf("Detail 'myn' Status: %d\n", resp2.StatusCode)
		fmt.Printf("Detail 'myn' Response: %s\n", string(body2))
	}
}
