package main

import (
	crawler "github.com/crawler/pkg"
)

// Todo: some basic CLI here
func main() {
	dcrawler := crawler.Crawler{Seeds: []string{""}}
	dcrawler.Crawl()
}
