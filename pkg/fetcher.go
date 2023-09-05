package hopper

import (
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

const (
	DefaultFetcherTimeout = 10 * time.Second
	DefaultFetcherDelay   = 15 * time.Second
)

type Fetcher struct {
	Client  *http.Client
	Delay   time.Duration
	Headers http.Header

	robots sync.Map
	groups sync.Map
}

func (f *Fetcher) Init() {
	if f.Client == nil {
		f.Client = http.DefaultClient
		f.Client.Timeout = DefaultFetcherTimeout
	}
	if f.Delay == 0 {
		f.Delay = DefaultFetcherDelay
	}

	f.Headers = http.Header{}
	f.robots = sync.Map{}
	f.groups = sync.Map{}
}

func (f *Fetcher) Do(method string, uri *url.URL, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		return nil, err
	}

    for k, v := range f.Headers {
		if headers.Get(k) == "" {
			headers[k] = v
		}
	}

	req.Header = headers

	return f.Client.Do(req)
}

func (f *Fetcher) FetchRobots(host string) (*robotstxt.RobotsData, error) {
	uri := url.URL{Scheme: "https", Host: host, Path: "robots.txt"}

	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}

	robots, err := robotstxt.FromResponse(res)
	if err != nil {
		return nil, err
	}

	return robots, err
}

func (f *Fetcher) SetGroup(host string, robots *robotstxt.RobotsData) {
	group := robots.FindGroup(f.Headers.Get("User-Agent"))
	f.groups.Store(host, group)
	f.robots.Store(host, robots)
}

func (f *Fetcher) GetGroup(host string, userAgent string) (*robotstxt.Group, bool) {
	if userAgent == f.Headers.Get("User-Agent") || userAgent == "" {
		group, exists := f.groups.Load(host)
		if !exists {
			return nil, exists
		}
		return group.(*robotstxt.Group), exists
	}

	robots, exists := f.robots.Load(host)
	if !exists {
		return nil, exists
	}
	return robots.(*robotstxt.RobotsData).FindGroup(userAgent), exists
}

func (f *Fetcher) Crawlable(uri *url.URL, userAgent string) bool {
	group, exists := f.GetGroup(uri.Hostname(), userAgent)
	if !exists {
		return true
	}

	return group.Test(uri.Path)
}

func (f *Fetcher) GetDelay(uri *url.URL, userAgent string) time.Duration {
	group, exists := f.GetGroup(uri.Hostname(), userAgent)
	if exists && group.CrawlDelay != 0 {
		return group.CrawlDelay
	}

	return f.Delay
}
