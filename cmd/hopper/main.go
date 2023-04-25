package main

import hopper "github.com/edwces/hopper/pkg"

// Todo: some basic CLI here
func main() {
	crawler := hopper.Crawler{Seeds: []string{}}
	crawler.Crawl()
}
