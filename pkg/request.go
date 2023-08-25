package hopper

import (
	"errors"
	"net/http"
	"net/url"
	"time"

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
	Document *html.Node

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

func (req *Request) Do() []*Request {
	defer req.End()

	req.BeforeRequest(req)
	req.Fetch()
	req.Parse()
	return req.Discover()
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
	req.Document = nil
	req.Depth++

	if !req.Valid() {
		return nil, errors.New("Invalid request")
	}

	return &req, nil
}

func (req *Request) Fetch() {
	req.BeforeFetch(req)
	defer req.AfterFetch(req)

	httpReq, err := http.NewRequest(req.Method, req.URL.String(), nil)
	if err != nil {
		return
	}

	for h, val := range req.Headers {
		httpReq.Header.Set(h, val)
	}

	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return
	}

    // Naive checking for content length as some website don't return this header
    // TODO: Implement MaxBytesReader on req.Body
    if httpRes.ContentLength != -1 && httpRes.ContentLength < req.Properties["ContentLength"].(int64) {
        return
    } 

	req.Response = httpRes
}

func (req *Request) Parse() {
	if req.Response == nil {
		return
	}

	req.BeforeParse(req)
	defer req.AfterParse(req)

	doc, err := html.Parse(req.Response.Body)
	if err != nil {
		return
	}
	req.Document = doc
}

func (req *Request) Discover() []*Request {
	discovered := []*Request{}
	var f func(*html.Node)

	if req.Document == nil {
		return discovered
	}

	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
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
	f(req.Document)

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
