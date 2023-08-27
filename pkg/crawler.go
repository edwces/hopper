package hopper

import (
	"math"
	"net/http"
	"sync"
	"time"
)

type Crawler struct {
	Concurrency       int
	Delay             time.Duration
	UserAgent         string
	AllowedDomains    []string
	DisallowedDomains []string
	AllowedDepth      int
	ContentLength     int64
	Client            *http.Client

	queue   *URLQueue
	request *Request

	BeforeRequest func(*Request)
	AfterRequest  func(*Request)
	BeforeParse   func(*Request)
	AfterParse    func(*Request)
	BeforeFetch   func(*Request)
	AfterFetch    func(*Request)

	OnError func(*Request, error)
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
	c.queue = &URLQueue{Max: c.Concurrency}
	c.queue.Init()
	c.request = &Request{BeforeRequest: c.BeforeRequest, AfterRequest: c.AfterRequest, BeforeFetch: c.BeforeFetch, AfterFetch: c.AfterFetch, BeforeParse: c.BeforeParse, AfterParse: c.AfterParse}
	c.request.Init()

	if c.OnError == nil {
		c.OnError = func(r *Request, err error) {}
	}

	c.request.Properties["Delay"] = c.Delay
	c.request.Properties["AllowedDomains"] = c.AllowedDomains
	c.request.Properties["DisallowedDomains"] = c.DisallowedDomains
	c.request.Properties["AllowedDepth"] = c.AllowedDepth
	c.request.Properties["ContentLength"] = c.ContentLength
	c.request.Headers["User-Agent"] = c.UserAgent
	c.request.Properties["RobotsMap"] = &sync.Map{}
    c.request.Properties["Client"] = c.Client

	if c.Delay == 0 {
		c.request.Properties["Delay"] = DefaultDelay
	}
	if c.UserAgent == "" {
		c.request.Headers["User-Agent"] = DefaultUserAgent
	}
	if c.AllowedDomains == nil {
		c.request.Properties["AllowedDomains"] = []string{}
	}
	if c.DisallowedDomains == nil {
		c.request.Properties["DisallowedDomains"] = []string{}
	}
	if c.AllowedDepth == 0 {
		c.request.Properties["AllowedDepth"] = math.MaxInt
	}
	if c.ContentLength == 0 {
		c.request.Properties["ContentLength"] = int64(4000000)
	}
    if c.Client == nil {
        client := http.DefaultClient
        client.Timeout = 5 * time.Second
        c.request.Properties["Client"] = client
    }

}

// Run is responsible for creating crawler workers.
func (c *Crawler) Run(seeds ...string) {
	if len(seeds) == 0 {
		panic("Cannot run crawler without seeds")
	}

	for _, seed := range seeds {
		req, err := c.request.New("GET", seed)
		if err != nil {
			continue
		}
		go c.queue.Push(req)
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
		req := c.queue.Pop()

		discovered, err := req.Do()

		if err != nil {
			c.OnError(req, err)
			continue
		}

		for _, discovery := range discovered {
			c.queue.Push(discovery)
		}
	}
}
