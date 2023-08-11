package hopper

import (
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// NOTE: might be better to refactor Worker proccess into it's own class
// but this way we would also need some easy way to copy config

const (
	DefaultUserAgent = "hopper/0.1"
	DefaultDelay     = time.Second * 15
)

type Crawler struct {
	sync.Mutex
	UserAgent string
	OnParse   func(*http.Response, *html.Node)
	Threads   int
	Delay     time.Duration

	queue *RequestQueue
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
	if c.UserAgent == "" {
		c.UserAgent = DefaultUserAgent
	}
	if c.Threads == 0 {
		c.Threads = runtime.GOMAXPROCS(0)
	}
	if c.Delay == 0 {
		c.Delay = DefaultDelay
	}
	if c.OnParse == nil {
		c.OnParse = func(r *http.Response, n *html.Node) {}
	}

	c.queue = NewRequestQueue(c.Threads)
}

// Run is responsible for creating crawler workers.
func (c *Crawler) Run(seeds ...string) {
	for _, seed := range seeds {
		if uri, err := url.Parse(seed); err == nil {
			go c.queue.Push(c.NewRequest(uri))
		}
	}

	for free := range c.queue.Free {
		for i := 0; i < free; i++ {
			go c.StartNewWorker()
		}
	}
}

// StartNewWorker starts a new work loop.
func (c *Crawler) StartNewWorker() {
	c.queue.AddThread()
	c.Traverse()
	c.queue.RemoveThread()
}

// Traverse starts crawl proccess until all links have been crawled.
func (c *Crawler) Traverse() {
	for c.queue.Len() != 0 {
		c.Visit(c.queue.Pop())
	}
}

// Visit proccesses given url
func (c *Crawler) Visit(uri *url.URL) {
	res, err := c.Fetch(uri)

	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	doc, err := html.Parse(strings.NewReader(string(bytes)))
	if err != nil {
		return
	}

	c.OnParse(res, doc)

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					discovery, err := url.Parse(attr.Val)
					if err != nil {
						continue
					}
					resolved := uri.ResolveReference(discovery)
					if !validURI(resolved) {
						continue
					}

					c.queue.Push(c.NewRequest(resolved))

				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			f(ch)
		}
	}
	f(doc)
}

// Fetch requests the uri and adds custom user defined headers.
func (c *Crawler) Fetch(uri *url.URL) (*http.Response, error) {
	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return res, err
	}

	return res, nil
}

// NewRequest creates new crawl request.
func (c *Crawler) NewRequest(uri *url.URL) *Request {
	return &Request{URI: uri, Delay: c.Delay}
}

func validURI(uri *url.URL) bool {
	if uri.Scheme != "http" && uri.Scheme != "https" {
		return false
	}
	return true
}
