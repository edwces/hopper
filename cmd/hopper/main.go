package main

import (
	"fmt"

	hopper "github.com/edwces/hopper/pkg"
)

// Todo: some basic CLI here
func main() {
	crawler := hopper.Crawler{}
    
    crawler.BeforeRequest = func(r *hopper.Request) {
        fmt.Println(r.URL.String())
    }

	crawler.Init()
	crawler.Run("https://en.wikipedia.org/wiki/Main_Page")
}
