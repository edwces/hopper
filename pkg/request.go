package hopper

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
	"golang.org/x/net/html"
)

const (
	DefaultUserAgent = "hopper/0.1"
	DefaultDelay     = time.Second * 15
)

type Request struct {
	Method string
	URL    *url.URL
	Depth  int

	Response *http.Response

	Headers    map[string]string
	Properties map[string]any

	BeforeRequest func(*Request)
	AfterRequest  func(*Request)
	BeforeParse   func(*Request)
	AfterParse    func(*Request)
	BeforeFetch   func(*Request)
	AfterFetch    func(*Request)
}

func (req *Request) Init() {
	if req.BeforeFetch == nil {
		req.BeforeFetch = func(r *Request) {}
	}
	if req.BeforeRequest == nil {
		req.BeforeRequest = func(r *Request) {}
	}
	if req.BeforeParse == nil {
		req.BeforeParse = func(r *Request) {}
	}

	if req.AfterFetch == nil {
		req.AfterFetch = func(r *Request) {}
	}
	if req.AfterRequest == nil {
		req.AfterRequest = func(r *Request) {}
	}
	if req.AfterParse == nil {
		req.AfterParse = func(r *Request) {}
	}

	req.URL = &url.URL{}
	req.Headers = map[string]string{}
	req.Properties = map[string]any{}
	req.Depth = -1
}

func (req *Request) Do() ([]*Request, error) {
	defer req.End()

    group, err := req.FetchRobots()
    if err != nil {
        return nil, err
    }
    if !group.Test(req.URL.Path) {
        return nil, errors.New("Robots.txt excluded path")
    }
    // Set specific delay for this request
    if group.CrawlDelay != 0 {
        req.Properties["Delay"] = group.CrawlDelay
    }

	req.BeforeRequest(req)

    res, err := req.Fetch()
    if err != nil {
        return nil, err
    }
    req.Response = res
    doc, err := req.Parse(res.Body)
    if err != nil {
        return nil, err
    }
	return req.Discover(doc), nil
} 

func (req Request) New(method string, uri string) (*Request, error) {
	parsed, err := req.URL.Parse(uri)
	if err != nil {
		return nil, err
        
	}
	if !parsed.IsAbs() {
		return nil, errors.New("Relative url can't be resolved")
	}
	parsed.Fragment = ""

	req.URL = parsed
	req.Method = method
	req.Response = nil
	req.Depth++

    // TEMP: Fix for delay, properties should probably be replaced with context
    // Because of this currently crawl options will not work

    // NOTE: Because properties are a map we are deep copying it to new request
    newProperties := map[string]any{}
    for k, v := range req.Properties {
        newProperties[k] = v
    }
    newProperties["Delay"] = DefaultDelay

	if !req.Valid() {
		return nil, errors.New("Invalid request")
	}

	return &req, nil
}

func (req *Request) FetchRobots() (*robotstxt.Group, error) {
    client := req.Properties["Client"].(*http.Client)
    robotsMap := req.Properties["RobotsMap"].(*sync.Map)
    group, exists := robotsMap.Load(req.URL.Hostname())
    if !exists {
        robotsURL := *req.URL
        robotsURL.JoinPath("robots.txt")

        httpReq, err := http.NewRequest(req.Method, req.URL.String(), nil)
        if err != nil {
            return nil, err
        }

        for h, val := range req.Headers {
            httpReq.Header.Set(h, val)
        }

        httpRes, err := client.Do(httpReq)
        if err != nil {
            return nil, err
        }
        robots, err := robotstxt.FromResponse(httpRes)
        if err != nil {
            return nil, err
        }
        group := robots.FindGroup(req.Headers["User-Agent"])
        robotsMap.Store(req.URL.Hostname(), group)        
        return group, nil
    }
    return group.(*robotstxt.Group), nil
}

func (req *Request) Fetch() (*http.Response, error) {
	req.BeforeFetch(req)
	defer req.AfterFetch(req)
    
    client := req.Properties["Client"].(*http.Client)
	httpReq, err := http.NewRequest(req.Method, req.URL.String(), nil)
	if err != nil {
		return nil, err
	}

	for h, val := range req.Headers {
		httpReq.Header.Set(h, val)
	}

	httpRes, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
    if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
        return nil, errors.New("Invalid status code: " + httpRes.Status)
    }

    return httpRes, nil
}

func (req *Request) Parse(body io.Reader) (*html.Node, error) {
	req.BeforeParse(req)
	defer req.AfterParse(req)

    // Naive checking for content length as some website don't return this header
    // TODO: Implement MaxBytesReader on req.Body
    reader := io.LimitReader(body, req.Properties["ContentLength"].(int64))

	return html.Parse(reader)
}

func (req *Request) Discover(node *html.Node) []*Request {
	discovered := []*Request{}
	var f func(*html.Node)

	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
                if attr.Key == "rel" && strings.Contains(attr.Val, "nofollow") {
                    continue
                }
				if attr.Key == "href" {
					resolved, err := req.New("GET", attr.Val)
					if err != nil {
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

func (req *Request) Valid() bool {
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return false
	}

	if req.Depth > req.Properties["AllowedDepth"].(int) {
		return false
	}

	for _, host := range req.Properties["AllowedDomains"].([]string) {
		if req.URL.Hostname() == host {
			continue
		}
		return false
	}

	for _, host := range req.Properties["DisallowedDomains"].([]string) {
		if req.URL.Hostname() != host {
			continue
		}
		return false
	}

	return true
}

func (req *Request) End() {
	if req.Response != nil {
		req.Response.Body.Close()
	}
	req.AfterRequest(req)
}
