package hopper

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Response struct {
    Body io.ReadCloser
    Req *Request
}

func (res *Response) Parse() (*html.Node, error) {
    bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(strings.NewReader(string(bytes)))
	if err != nil {
		return doc, err
	}

    return doc, err
}

func (res *Response) Discover(node *html.Node) []*url.URL {
    discovered := []*url.URL{}

    var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					discovery, err := url.Parse(attr.Val)
					if err != nil {
						continue
					}
					resolved := res.Req.URL.ResolveReference(discovery)
					if !validURI(resolved) {
						continue
					}
                    discovered = append(discovered, resolved)
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			f(ch)
		}
	}
	f(node)
    
    return discovered
}

func (res *Response) Close() {
   res.Body.Close()      
}

func validURI(uri *url.URL) bool {
	if uri.Scheme != "http" && uri.Scheme != "https" {
		return false
	}
	return true
}
