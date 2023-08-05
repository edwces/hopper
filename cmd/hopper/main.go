package main

import (
	"fmt"
	"net/http"

	hopper "github.com/edwces/hopper/pkg"
	"golang.org/x/net/html"
)

// Todo: some basic CLI here
func main() {
	crawler := hopper.Crawler{}

    crawler.OnParse = func(res *http.Response, n *html.Node) {
        fmt.Println(res.Request.URL.String())
    }

	crawler.Init()
	crawler.Traverse("https://crawler-test.com/redirects/redirect_1")
}
