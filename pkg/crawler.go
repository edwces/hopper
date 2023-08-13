package hopper

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// NOTE: might be better to refactor Worker proccess into it's own class
// but this way we would also need some easy way to copy config

type Crawler struct {
	sync.Mutex
	UserAgent string
	OnParse   func(*http.Response, *html.Node)
	Threads   int
	Delay     time.Duration

	queue *URLQueue
    request *Request
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
	if c.OnParse == nil {
		c.OnParse = func(r *http.Response, n *html.Node) {}
	}

    c.queue = &URLQueue{max: c.Threads}
    c.queue.Init()
    c.request = &Request{UserAgent: c.UserAgent, Delay: c.Delay}
    c.request.Init()
}

// Run is responsible for creating crawler workers.
func (c *Crawler) Run(seeds ...string) {
	for _, seed := range seeds {
		if uri, err := url.Parse(seed); err == nil {
			go c.queue.Push(uri, c.request.Delay)
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
        req := c.request.New("GET", c.queue.Pop())
		c.Visit(req)
	}
}

// Visit proccesses given url
func (c *Crawler) Visit(req *Request) {
	res, err := req.Do()

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
					resolved := req.URL.ResolveReference(discovery)
					if !validURI(resolved) {
						continue
					}

					c.queue.Push(resolved, c.request.Delay)

				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			f(ch)
		}
	}
	f(doc)
}

func validURI(uri *url.URL) bool {
	if uri.Scheme != "http" && uri.Scheme != "https" {
		return false
	}
	return true
}
