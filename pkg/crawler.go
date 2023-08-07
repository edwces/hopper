package hopper

import (
	"net/http"
	"net/url"
	"runtime"

	"golang.org/x/net/html"
)

const DefaultUserAgent = "hopper/0.1"

type Crawler struct {
    UserAgent string
	OnParse func(*http.Response, *html.Node)
    Threads int

	queue *URLQueue
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
    if c.UserAgent == "" {
        c.UserAgent = DefaultUserAgent
    }
    if c.Threads == 0 {
        c.Threads = runtime.GOMAXPROCS(0)
    }

    if c.OnParse == nil {
        c.OnParse = func(r *http.Response, n *html.Node) {}
    }

	c.queue = NewURLQueue(c.Threads)
}

// Traverse uses depth-first search for link traversal.
func (c *Crawler) Run(seeds ...string) {
	for _, seed := range seeds {
		if uri, err := url.Parse(seed); err == nil {
             go c.queue.Push(uri)
		}
	}
    
    // For each free place in queue create new Worker
    for free := range c.queue.Free {
        for i := 0; i < free; i++ {
            go c.Work()
        }
    }
}

// NOTE: Figure some way to not copy all of config every time
func (c *Crawler) Work() {
    c.queue.AddThread()
    work := Worker{queue: c.queue, OnParse: c.OnParse, UserAgent: c.UserAgent}
    work.Traverse()
    c.queue.RemoveThread()
}


