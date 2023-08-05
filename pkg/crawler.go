package hopper

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const DefaultUserAgent = "hopper/0.1"

type Crawler struct {
    UserAgent string
	OnParse func(*http.Response, *html.Node)

	queue *URLQueue 
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
    if c.UserAgent == "" {
        c.UserAgent = DefaultUserAgent
    }

    if c.OnParse == nil {
        c.OnParse = func(r *http.Response, n *html.Node) {}
    }

	c.queue = NewURLQueue() 
}

// Traverse uses depth-first search for link traversal.
func (c *Crawler) Traverse(seeds ...string) {
	for _, seed := range seeds {
		if uri, err := url.Parse(seed); err == nil {
            c.queue.Push(uri)
		}
	}

	for c.queue.Length() != 0 {
		c.Visit(c.queue.Pop())
	}
}

func (c *Crawler) Visit(uri *url.URL) {
	res, err := c.Request(uri.String())
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

	// Extract uri's from document.
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
                    c.queue.Push(resolved)
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			f(ch)
		}
	}
	f(doc)
}

// Send a http GET Request with defined headings and return response. 
func (c *Crawler) Request(uri string) (*http.Response, error) {
    req, err := http.NewRequest("GET", uri, nil)
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

func validURI(uri *url.URL) bool {
	if uri.Scheme != "http" && uri.Scheme != "https" {
		return false
	}
	return true
}
