package hopper

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

const (
    DefaultFetcherTimeout = 10 * time.Second
    DefaultFetcherDelay = 15 * time.Second
)

type Fetcher struct {
    Client *http.Client
    Delay time.Duration
    Headers map[string]string

    robots sync.Map
}

func (f *Fetcher) Init() {
    if f.Client == nil {
        f.Client = http.DefaultClient
        f.Client.Timeout = DefaultFetcherTimeout
    }
    if f.Delay == 0 {
        f.Delay = DefaultFetcherDelay
    }

    f.Headers = map[string]string{}
    f.robots = sync.Map{}
}

func (f *Fetcher) SetDefaultHeaders(r *Request) {
    for k, v := range f.Headers {
        if r.Headers.Get(k) != "" {
            r.Headers.Set(k, v)
        }
    }
}

func (f *Fetcher) Do(r *Request) (*http.Response, error) {
    req, err := http.NewRequest(r.Method, r.URL.String(), nil)
    if err != nil {
        return nil, err
    }

    req.Header = r.Headers

    return f.Client.Do(req)
}

func (f *Fetcher) FetchHTML(r *Request) (*http.Response, error) {
	res, err := f.Do(r)
	if err != nil {
		return nil, err
	}

    if res.StatusCode < 200 || res.StatusCode >= 300 {
        return nil, errors.New("Invalid status code: " + res.Status)
    }

    return res, err
}

func (f *Fetcher) FetchRobots(r *Request) (*robotstxt.Group, error) {
    group, exists := f.robots.Load(r.URL.Hostname())
    if !exists {
        req, err := r.New(http.MethodGet, "/robots.txt")
        if err != nil {
            return nil, err
        }

        res, err := f.Do(req)
        if err != nil {
            return nil, err
        }

        robots, err := robotstxt.FromResponse(res)
        if err != nil {
            return nil, err
        }

        group = robots.FindGroup(r.Headers.Get("User-Agent"))
        f.robots.Store(r.URL.Hostname(), group)        
    }

    return group.(*robotstxt.Group), nil
}
