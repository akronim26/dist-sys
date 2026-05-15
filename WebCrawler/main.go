package main

import (
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
	fmt.Println(websiteName, crawlingLimit)
}
