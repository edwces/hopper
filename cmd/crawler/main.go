package main

import (
	crawler "github.com/crawler/pkg"
)

// Todo: some basic CLI here
func main() {
	crawler.Crawl([]string{}, []string{"*"}, []string{})
}
