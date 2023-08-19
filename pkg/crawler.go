package hopper

import (
	"net/url"
	"time"

	"golang.org/x/net/html"
)

// NOTE: might be better to refactor Worker proccess into it's own class
// but this way we would also need some easy way to copy config

type Crawler struct {
	UserAgent string
	OnParse   func(*Response, *html.Node)
	Threads   int
	Delay     time.Duration

	queue *URLQueue
    request *Request
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
	if c.OnParse == nil {
		c.OnParse = func(r *Response, n *html.Node) {}
	}

    c.queue = &URLQueue{Max: c.Threads}
    c.queue.Init()
    c.request = &Request{UserAgent: c.UserAgent, Delay: c.Delay}
    c.request.Init()
    
}

// Run is responsible for creating crawler workers.
func (c *Crawler) Run(seeds ...string) {
    if len(seeds) == 0 {
        panic("Cannot run crawler without seeds")
    }

	for _, seed := range seeds {
		if uri, err := url.Parse(seed); err == nil {
			go c.queue.Push(uri, c.request.Delay)
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
		c.Visit(req)
	}
}

// Visit proccesses given url
func (c *Crawler) Visit(req *Request) {
	httpRes, err := req.Do()
	if err != nil {
		return
	}
    
    res := &Response{Body: httpRes.Body, Req: req}
    defer res.Close()

    doc, err := res.Parse()
	if err != nil {
		return
    }

	c.OnParse(res, doc)

    discovered := res.Discover(doc)
    for _, discovery := range discovered {
        go c.queue.Push(discovery, req.Delay)
    }
    
}


