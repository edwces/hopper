package hopper

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

type (
	RequestHandler  = func(*Request) error
	ResponseHandler = func(*Response) error
	PushHandler     = func(*Request) error
	ErrorHandler    = func(*Request, error)
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

	onRequest  []RequestHandler
	onPush     []PushHandler
	onResponse []ResponseHandler
	onError    []ErrorHandler
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
	c.queue = &URLQueue{Max: c.Concurrency}
	c.queue.Init()

	fetcher := &Fetcher{Client: c.Client}
	fetcher.Init()
	fetcher.Headers.Set("User-Agent", c.UserAgent)

	c.request = &Request{fetcher: fetcher}
	c.request.Init()
	c.request.Properties["AllowedDomains"] = c.AllowedDomains
	c.request.Properties["DisallowedDomains"] = c.DisallowedDomains
	c.request.Properties["AllowedDepth"] = c.AllowedDepth

	if c.UserAgent == "" {
		fetcher.Headers.Set("User-Agent", DefaultUserAgent)
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
    if int(c.Delay) == 0 {
        c.Delay = DefaultDelay
    }

	c.onRequest = []RequestHandler{}
	c.onResponse = []ResponseHandler{}
	c.onPush = []PushHandler{}
	c.onError = []ErrorHandler{}
}

func (c *Crawler) OnRequest(fn RequestHandler) {
	c.onRequest = append([]RequestHandler{fn}, c.onRequest...)
}

func (c *Crawler) OnResponse(fn ResponseHandler) {
	c.onResponse = append([]ResponseHandler{fn}, c.onResponse...)
}

func (c *Crawler) OnPush(fn PushHandler) {
	c.onPush = append([]PushHandler{fn}, c.onPush...)
}

func (c *Crawler) OnError(fn ErrorHandler) {
	c.onError = append([]ErrorHandler{fn}, c.onError...)
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
		go c.Push(req)
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

		err := c.Visit(req)
		if err != nil {
			for _, fn := range c.onError {
				fn(req, err)
			}
		}
	}
}

func (c *Crawler) Visit(req *Request) error {
	for _, fn := range c.onRequest {
		err := fn(req)
		if err != nil {
			return fmt.Errorf("Request: %w", err)

		}
	}

	httpRes, err := req.Do()
	if err != nil {
		return fmt.Errorf("Request: %w", err)
	}

	res, err := NewResponse(httpRes, req.Properties, req)
	if err != nil {
		return fmt.Errorf("Response: %w", err)
	}

	for _, fn := range c.onResponse {
		err := fn(res)
		if err != nil {
			return fmt.Errorf("Response: %w", err)

		}
	}

	discovered, err := res.Do()
	if err != nil {
		return fmt.Errorf("Response: %w", err)
	}

	for _, discovery := range discovered {
        err := c.Push(discovery)
        if err != nil {
            for _, fn := range c.onError {
                fn(discovery, err)
            }
        }
	}

	return nil
}

func (c *Crawler) Push(req *Request) error {
    for _, fn := range c.onPush {
        err := fn(req)
        if err != nil {
            return fmt.Errorf("Push: %w", err)
        }
	}
    
    // Temp fix for default delay for queue
    delay, exist := req.Properties["Delay"]
    if !exist || int(delay.(time.Duration)) == 0 {
        req.Properties["Delay"] = c.Delay 
    }

    c.queue.Push(req)
    return nil
}
