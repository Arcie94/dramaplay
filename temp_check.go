package main

import (
	"fmt"
	"io"
	"net/http"
)

const Token = "0ebd6cfdd8054d2a90aa2851532645211aeaf189fa1aed62c53e5fd735af8649"

func check(url string) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+Token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("URL: %s\nStatus: %d\nLength: %d\nBody: %s\n\n", url, resp.StatusCode, len(body), string(body))
}

func main() {
	fmt.Println("Checking FreeShort...")
	check("https://sapimu.au/freeshort/api/v1/dramas/new?lang=4")
	check("https://sapimu.au/freeshort/api/v1/dramas/new?lang=3")
}
