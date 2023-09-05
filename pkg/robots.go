package hopper

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

type RobotsTxtMiddleware struct {
    Client *Fetcher

    groups sync.Map
    robots sync.Map
}

func RobotsTxt(crawler *Crawler) {
    rt := &RobotsTxtMiddleware{Client: crawler.request.fetcher}

    crawler.OnRequest(func(r *Request) error {
        _, exists := rt.GetGroup(r.URL.Hostname(), r.Headers.Get("User-Agent"))
        if !exists {
            robots, err := rt.Fetch(r.URL.Hostname())
            if err != nil {
                return err
            }
            rt.SetGroup(r.URL.Hostname(), robots)
        }

        if !rt.Crawlable(r.URL, r.Headers.Get("User-Agent")) {
            return errors.New("Robots.txt excluded path")
        }

        return nil
    })

    crawler.OnPush(func(r *Request) error {
        if !rt.Crawlable(r.URL, r.Headers.Get("User-Agent")) {
            return errors.New("Robots.txt excluded path")
        }

        r.Properties["Delay"] = rt.GetDelay(r.URL, r.Headers.Get("User-Agent"))

        return nil
    })
}

func (rt *RobotsTxtMiddleware) Fetch(host string) (*robotstxt.RobotsData, error) {
	uri := url.URL{Scheme: "https", Host: host, Path: "robots.txt"}

	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := rt.Client.Do(req.Method, req.URL, nil, req.Header)
	if err != nil {
		return nil, err
	}

	robots, err := robotstxt.FromResponse(res)
	if err != nil {
		return nil, err
	}

	return robots, err
}

func (rt *RobotsTxtMiddleware) SetGroup(host string, robots *robotstxt.RobotsData) {
	group := robots.FindGroup(rt.Client.Headers.Get("User-Agent"))
	rt.groups.Store(host, group)
	rt.robots.Store(host, robots)
}

func (rt *RobotsTxtMiddleware) GetGroup(host string, userAgent string) (*robotstxt.Group, bool) {
	if userAgent == rt.Client.Headers.Get("User-Agent") || userAgent == "" {
		group, exists := rt.groups.Load(host)
		if !exists {
			return nil, exists
		}
		return group.(*robotstxt.Group), exists
	}

	robots, exists := rt.robots.Load(host)
	if !exists {
		return nil, exists
	}
	return robots.(*robotstxt.RobotsData).FindGroup(userAgent), exists
}

func (rt *RobotsTxtMiddleware) Crawlable(uri *url.URL, userAgent string) bool {
	group, exists := rt.GetGroup(uri.Hostname(), userAgent)
	if !exists {
		return true
	}

	return group.Test(uri.Path)
}

func (rt *RobotsTxtMiddleware) GetDelay(uri *url.URL, userAgent string) time.Duration {
	group, exists := rt.GetGroup(uri.Hostname(), userAgent)
	if !exists {
		return 0
	}

	return group.CrawlDelay
}

