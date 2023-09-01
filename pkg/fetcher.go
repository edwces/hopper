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

		f.SetRobots(req.URL.Hostname(), robots)
	}
	group, _ := f.robots.Load(r.URL.Hostname())

	return group.(*robotstxt.Group), nil
}

func (f *Fetcher) SetRobots(host string, robots *robotstxt.RobotsData) {
	group := robots.FindGroup(f.Headers["User-Agent"])
	f.robots.Store(host, group)
}

func (f *Fetcher) Crawlable(uri *url.URL) bool {
	group, exists := f.robots.Load(uri.Hostname())
	if exists && !group.(*robotstxt.Group).Test(uri.Path) {
		return false
	}
	return true
}

func (f *Fetcher) GetDelay(uri *url.URL) time.Duration {
	group, exists := f.robots.Load(uri.Hostname())
	if exists && group.(*robotstxt.Group).CrawlDelay != 0 {
		return group.(*robotstxt.Group).CrawlDelay
	}
	return f.Delay
}
