package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run . <webiste_url> <crawling_limit>")
		return
	}

	websiteName := os.Args[1]
	crawlingLimit, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Crawling limit should be a number")
		return
	}

	crawler := NewCrawler(crawlingLimit)
	result := crawler.Crawl(websiteName)

	fmt.Printf("\nCrawling results for %s (Limit: %d):\n", websiteName, crawlingLimit)
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}
