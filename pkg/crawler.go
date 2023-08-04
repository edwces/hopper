package hopper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Crawler struct {
	queue     []*url.URL
	seen      map[*url.URL]bool
}

// Init initializes default values for crawler.
func (c *Crawler) Init() {
    c.queue = []*url.URL{}
    c.seen = map[*url.URL]bool{}
}

// Traverse uses depth-first search for link traversal.
func (c *Crawler) Traverse(seeds ...string) {
	for _, seed := range seeds {
		if uri, err := url.Parse(seed); err == nil {
        	c.seen[uri] = true
		    c.queue = append(c.queue, uri)
        }
	}

	for len(c.queue) != 0 {
        uri := c.queue[len(c.queue)-1]
        c.queue = c.queue[:len(c.queue)-1]
		c.Visit(uri)
	}
}

func (c *Crawler) Visit(uri *url.URL) {
    fmt.Println("URL: ", uri.String())
    resp, err := http.Get(uri.String())
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return 
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 
	}
	doc, err := html.Parse(strings.NewReader(string(bytes)))
	if err != nil {
		return 
	}
    
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
                    if c.seen[resolved] || !validURI(resolved) {
                        continue
                    }
                    c.seen[resolved] = true
		            c.queue = append(c.queue, discovery)
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
