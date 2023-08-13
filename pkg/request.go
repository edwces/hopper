package hopper

import (
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultUserAgent = "hopper/0.1"
	DefaultDelay     = time.Second * 15
)

type Request struct {
    Method string
	URL *url.URL

    UserAgent string
    Delay time.Duration
}

func (req *Request) Init() {
    if req.UserAgent == "" {
        req.UserAgent = DefaultUserAgent
    }
    if req.Delay == 0 {
		req.Delay = DefaultDelay
	}
}

func (req Request) New(method string, uri *url.URL) *Request {
    req.URL = uri
    req.Method = method
    return &req
}

func (req *Request) Do() (*http.Response, error) {
    httpReq, err := http.NewRequest(req.Method, req.URL.String(), nil)
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("User-Agent", req.UserAgent)

    return http.DefaultClient.Do(httpReq)
}


