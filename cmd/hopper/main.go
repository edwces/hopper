package main

import hopper "github.com/edwces/hopper/pkg"

// Todo: some basic CLI here
func main() {
	crawler := hopper.Crawler{}
	crawler.Init()
	crawler.Crawl("")
}
