package hopper

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	DefaultUserAgent = "hopper/0.1"
	DefaultDelay     = time.Second * 15
)

type Request struct {
    Method string
	URL *url.URL

    Response *http.Response
    Document *html.Node

    Headers map[string]string
    Properties map[string]any

    BeforeRequest func (*Request)
    AfterRequest func (*Request)
    BeforeParse func (*Request)
    AfterParse func (*Request)
    BeforeFetch func (*Request)
    AfterFetch func (*Request)
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

    req.Headers = map[string]string{}
    req.Properties = map[string]any{}
}


func (req *Request) Do() []*Request {
    defer req.End()
    
    req.BeforeRequest(req)
    req.Fetch()
    req.Parse()
    return req.Discover()
}

func (req Request) New(method string, uri *url.URL) *Request {
    req.URL = uri
    req.Method = method
    return &req
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
    req.Response = httpRes
}

func (req *Request) Parse() {
    if req.Response == nil {
        return
    }

    req.BeforeParse(req)
    defer req.AfterParse(req)

    bytes, err := io.ReadAll(req.Response.Body)
	if err != nil {
		return
	}
	doc, err := html.Parse(strings.NewReader(string(bytes)))
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
					discovery, err := url.Parse(attr.Val)
					if err != nil {
						continue
					}
					resolved := req.URL.ResolveReference(discovery)
                    discovered = append(discovered, req.New("GET", resolved))
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

func (req *Request) End() {
    if req.Response != nil {
        req.Response.Body.Close()
    }
    req.AfterRequest(req)
}
