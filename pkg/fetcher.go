package hopper

import (
	"errors"
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

// Add Some option to robots option so for persistentUserAgent so our Queries can be optimized
// by having two fields for robots and groups for default user-agent
// BUG: For example for now when user will set different User-agent for request robots txt
// group will be saved and used only for that user-agent

type Fetcher struct {
	Client  *http.Client
	Delay   time.Duration
	Headers map[string]string

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

	f.Headers = map[string]string{}
	f.robots = sync.Map{}
	f.groups = sync.Map{}
}

func (f *Fetcher) Do(r *Request) (*http.Response, error) {
	req, err := http.NewRequest(r.Method, r.URL.String(), nil)
	if err != nil {
		return nil, err
	}

	for k, v := range f.Headers {
		if r.Headers.Get(k) != "" {
			r.Headers.Set(k, v)
		}
	}

	req.Header = r.Headers

	return f.Client.Do(req)
}

func (f *Fetcher) Fetch(r *Request) (*http.Response, error) {
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
	_, exists := f.robots.Load(r.URL.Hostname())
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

		f.SetGroup(req.URL.Hostname(), robots)
	}
	group, _ := f.getGroup(r.URL.Hostname(), r.Headers.Get("User-Agent"))

	return group, nil
}

func (f *Fetcher) SetGroup(host string, robots *robotstxt.RobotsData) {
	group := robots.FindGroup(f.Headers["User-Agent"])
	f.groups.Store(host, group)
	f.robots.Store(host, robots)
}

func (f *Fetcher) getGroup(host string, userAgent string) (*robotstxt.Group, bool) {
	if userAgent == f.Headers["User-Agent"] || userAgent == "" {
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
	group, exists := f.getGroup(uri.Hostname(), userAgent)
	if !exists {
		return true
	}

	return group.Test(uri.Path)
}

func (f *Fetcher) GetDelay(uri *url.URL, userAgent string) time.Duration {
	group, exists := f.getGroup(uri.Hostname(), userAgent)
	if exists && group.CrawlDelay != 0 {
		return group.CrawlDelay
	}

	return f.Delay
}
