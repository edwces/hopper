package main

import (
	"fmt"

	hopper "github.com/edwces/hopper/pkg"
	"golang.org/x/net/html"
)

// Todo: some basic CLI here
func main() {
	crawler := hopper.Crawler{}

	crawler.OnParse = func(res *hopper.Response, n *html.Node) {
		fmt.Println(res.Req.URL.String())
	}

	crawler.Init()
	crawler.Run("https://en.wikipedia.org/wiki/Main_Page")
}
