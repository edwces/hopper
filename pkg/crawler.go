package hopper

import (
	"fmt"
	"net/url"
	"time"
)

// NOTE: might be better to refactor Worker proccess into it's own class
// but this way we would also need some easy way to copy config

type Crawler struct {
	Threads   int

	queue *URLQueue
    request *Request
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
    c.queue = &URLQueue{Max: c.Threads}
    c.queue.Init()
    c.request = &Request{BeforeRequest: func(r *Request) {fmt.Println(r.URL.String())}}
    c.request.Init()
    
}

// Run is responsible for creating crawler workers.
func (c *Crawler) Run(seeds ...string) {
    if len(seeds) == 0 {
        panic("Cannot run crawler without seeds")
    }

	for _, seed := range seeds {
		if uri, err := url.Parse(seed); err == nil {
			go c.queue.Push(uri, c.request.Properties["Delay"].(time.Duration))
		}
	}

	for free := range c.queue.Free {
		for i := 0; i < free; i++ {
			go c.Traverse()
		}
	}
}

// Traverse starts crawl proccess until all links have been crawled.
func (c *Crawler) Traverse() {
	for c.queue.Len() != 0 {
        req := c.request.New("GET", c.queue.Pop())
        discovered := req.Do()

        for _, discovery := range discovered {
            c.queue.Push(discovery.URL, discovery.Properties["Delay"].(time.Duration))
        }
	}
}
