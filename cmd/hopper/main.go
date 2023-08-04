package main

import (
	"fmt"

	hopper "github.com/edwces/hopper/pkg"
)

// Todo: some basic CLI here
func main() {
    fmt.Println("Program started")
	crawler := hopper.Crawler{}
	crawler.Init()
	crawler.Traverse("")
    fmt.Println("Program ended")
}
