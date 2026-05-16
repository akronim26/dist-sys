package main

import (
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/PuerkitoBio/goquery"
	"github.com/PuerkitoBio/purell"
)

type Page struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type CrawlerResult struct {
	Pages []Page `json:"pages"`
}

type Crawler struct {
	Visited       map[string]bool
	PagesVisited  atomic.Int32
	ActiveWorkers atomic.Int32
	Queue         []string
	Limit         int
	QueueMu       sync.Mutex
	MapMu         sync.Mutex
	wg            sync.WaitGroup
	Pages         []Page
	PageMu        sync.Mutex
	cond          *sync.Cond
	client        *http.Client
}

func NewCrawler(crawlingLimit int) *Crawler {
	crawler := &Crawler{
		Visited:      make(map[string]bool),
		PagesVisited: atomic.Int32{},
		Queue:        []string{},
		Limit:        crawlingLimit,
		client:       &http.Client{},
	}
	crawler.cond = sync.NewCond(&crawler.QueueMu)
	return crawler
}

func Worker(crawler *Crawler, item string) {
	defer crawler.wg.Done()
	defer crawler.ActiveWorkers.Add(-1)
	defer crawler.cond.Broadcast()

	normalized, err := normalizeURL(item)
	if err != nil {
		log.Printf("failed to normalize url %s: %v", item, err)
		return
	}

	crawler.MapMu.Lock()
	if crawler.Visited[normalized] {
		crawler.MapMu.Unlock()
		return
	}
	crawler.Visited[normalized] = true
	crawler.MapMu.Unlock()

	req, err := http.NewRequest("GET", normalized, nil)
	if err != nil {
		log.Printf("error creating request for %s: %v", normalized, err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WebCrawler/1.0)")

	res, err := crawler.client.Do(req)
	if err != nil {
		log.Printf("error fetching %s: %v", normalized, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Printf("status code error for %s: %d %s", normalized, res.StatusCode, res.Status)
		return
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("error parsing %s: %v", normalized, err)
		return
	}

	title := doc.Find("title").Text()
	currentPage := Page{
		URL:   normalized,
		Title: title,
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

			crawler.QueueMu.Lock()
			crawler.Queue = append(crawler.Queue, absURL)
			crawler.QueueMu.Unlock()
			crawler.cond.Signal()
		}
	})

	crawler.PageMu.Lock()
	if len(crawler.Pages) < crawler.Limit {
		crawler.Pages = append(crawler.Pages, currentPage)
		crawler.PagesVisited.Add(1)
	}
	crawler.PageMu.Unlock()
}

func (c *Crawler) Crawl(websiteName string) *CrawlerResult {
	c.QueueMu.Lock()
	c.Queue = append(c.Queue, websiteName)
	c.QueueMu.Unlock()

	// Use a semaphore to limit max concurrent requests
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	for {
		c.QueueMu.Lock()
		for len(c.Queue) == 0 && c.ActiveWorkers.Load() > 0 && c.PagesVisited.Load() < int32(c.Limit) {
			c.cond.Wait()
		}

		// Termination conditions:
		// 1. Reached the limit
		// 2. Queue is empty AND no active workers are left (no more links possible)
		if c.PagesVisited.Load() >= int32(c.Limit) || (len(c.Queue) == 0 && c.ActiveWorkers.Load() == 0) {
			c.QueueMu.Unlock()
			break
		}

		item := c.Queue[0]
		c.Queue = c.Queue[1:]
		c.QueueMu.Unlock()

		sem <- struct{}{}
		c.wg.Add(1)
		c.ActiveWorkers.Add(1)

		go func(url string) {
			defer func() { <-sem }()
			Worker(c, url)
		}(item)
	}

	c.wg.Wait()
	return &CrawlerResult{
		Pages: c.Pages,
	}
}

func normalizeURL(url string) (string, error) {
	normalized, err := purell.NormalizeURLString(
		url, purell.FlagsUsuallySafeGreedy,
	)

	return normalized, err
}
