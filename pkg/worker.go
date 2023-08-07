package hopper

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)


type Worker struct {
    UserAgent string
	OnParse func(*http.Response, *html.Node)

    queue *URLQueue
}

func (w *Worker) Traverse() {
    for w.queue.Length() != 0 {
        w.Visit(w.queue.Pop())
    }
}

func (w *Worker) Visit(uri *url.URL) {
    res, err := w.Request(uri.String())
    // temporarily FOR safety
    time.Sleep(time.Second * 5)

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
    
	w.OnParse(res, doc)

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
                    
                    w.queue.Push(resolved)
                    
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			f(ch)
		}
	}
	f(doc)
}

func (w *Worker) Request(uri string) (*http.Response, error) {
    req, err := http.NewRequest("GET", uri, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("User-Agent", w.UserAgent)
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
