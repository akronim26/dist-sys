package main

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
}

func NewCralwer(crawlingLimit int) *Crawler {
	return &Crawler{
		Visited:      make(map[string]bool),
		PagesVisited: 0,
		Queue:        []string{},
		Limit:        crawlingLimit,
	}
}

func (c *Crawler) Crawl(websiteName string) *CrawlerResult {
	pages := []Page{}
	return &CrawlerResult{
		pages,
	}
}
