package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/PuerkitoBio/purell"
)

type Page struct {
	URL   string   `json:"url"`
	Title string   `json:"title"`
	Links []string `json:"links"`
}

type CrawlerResult struct {
	Pages []Page `json:"pages"`
}

type Crawler struct {
	Visited      map[string]bool
	PagesVisited int
	Queue        []string
	Limit        int
	QueueMu      sync.Mutex
	MapMu        sync.Mutex
}

func NewCrawler(crawlingLimit int) *Crawler {
	return &Crawler{
		Visited:      make(map[string]bool),
		PagesVisited: 0,
		Queue:        []string{},
		Limit:        crawlingLimit,
	}
}

func (c *Crawler) Crawl(websiteName string) *CrawlerResult {
	pages := []Page{}
	c.Queue = append(c.Queue, websiteName)

	for c.PagesVisited != c.Limit && len(c.Queue) > 0 {

		c.QueueMu.Lock()
		item := c.Queue[0]
		c.Queue = c.Queue[1:]
		c.QueueMu.Unlock()
		normalized, err := normalizeURL(item)
		if err != nil {
			log.Printf("failed to normalize url %s: %v", item, err)
			continue
		}

		c.MapMu.Lock()
		if c.Visited[normalized] {
			c.MapMu.Unlock()
			continue
		}
		c.Visited[normalized] = true

		c.MapMu.Unlock()

		client := &http.Client{}
		req, err := http.NewRequest("GET", normalized, nil)
		if err != nil {
			log.Printf("error creating request for %s: %v", normalized, err)
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WebCrawler/1.0)")

		res, err := client.Do(req)
		if err != nil {
			log.Printf("error fetching %s: %v", normalized, err)
			continue
		}
		if res.StatusCode != 200 {
			log.Printf("status code error for %s: %d %s", normalized, res.StatusCode, res.Status)
			res.Body.Close()
			continue
		}

		// Load the HTML document
		doc, err := goquery.NewDocumentFromReader(res.Body)
		res.Body.Close()
		if err != nil {
			log.Printf("error parsing %s: %v", normalized, err)
			continue
		}

		title := doc.Find("title").Text()
		currentPage := Page{
			URL:   normalized,
			Title: title,
			Links: []string{},
		}

		parsedBase, _ := url.Parse(normalized)
		// Find the links
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists {
				u, err := url.Parse(href)
				if err != nil {
					return
				}
				absURL := parsedBase.ResolveReference(u).String()

				c.QueueMu.Lock()
				c.Queue = append(c.Queue, absURL)
				c.QueueMu.Unlock()
				currentPage.Links = append(currentPage.Links, absURL)
				fmt.Printf("Link found: %s\n", absURL)
			}
		})
		pages = append(pages, currentPage)
		c.PagesVisited++
	}
	return &CrawlerResult{
		Pages: pages,
	}
}

func normalizeURL(url string) (string, error) {
	normalized, err := purell.NormalizeURLString(
		url, purell.FlagsUsuallySafeGreedy,
	)

	return normalized, err
}
