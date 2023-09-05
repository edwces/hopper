package hopper

import (
	"net/url"
	"testing"
	"time"

	"github.com/temoto/robotstxt"
)

const (
    TestRobotsDelay = 10 * time.Second
    TestUserAgent = "hopper/test"
)

func TestGetDelay(t *testing.T) {
    f := &Fetcher{}
    f.Init()
    f.Headers.Set("User-Agent", TestUserAgent)

    rt := RobotsTxtMiddleware{Client: f}

    uri, err := url.Parse("http://mock.com/hello")
    if err != nil {
        t.Fail()
    }
    
    t.Run("WithoutGroup", func(t *testing.T) {
        delay := rt.GetDelay(uri, "")

        if delay != time.Duration(0) {
            t.Fatalf("f.GetDelay = %s, want %s", delay, time.Duration(0))
        }
    })

    t.Run("WithoutGroupDelay", func(t *testing.T) {
        robots, err := robotstxt.FromString(`
            User-agent: hopper/test
            Disallow: /w/

            Sitemap: https://www.example.com/sitemap.xml
        `)
        if err != nil {
            t.Fail()
        }

        rt.SetGroup(uri.Hostname(), robots)
        delay := rt.GetDelay(uri, "")
        
        if delay != time.Duration(0) {
            t.Fatalf("f.GetDelay = %s, want %s", delay, time.Duration(0))
        }
    })

    t.Run("WithGroupDelay", func(t *testing.T) {
        robots, err := robotstxt.FromString(`
            User-agent: hopper/test
            Crawl-Delay: 10
            Disallow: /w/

            Sitemap: https://www.example.com/sitemap.xml
        `)
        if err != nil {
            t.Fail()
        }

        rt.SetGroup(uri.Hostname(), robots)
        delay := rt.GetDelay(uri, "")

        if delay != TestRobotsDelay {
            t.Fatalf("f.GetDelay = %s, want %s", delay, TestRobotsDelay)
        }
    })

    t.Run("WithGroupDelayOnDifferentAgent", func(t *testing.T) {
        robots, err := robotstxt.FromString(`
            User-agent: hopper/different
            Crawl-Delay: 10
            Disallow: /i/

            User-agent: hopper/test
            Disallow: /w/

            Sitemap: https://www.example.com/sitemap.xml
        `)
        if err != nil {
            t.Fail()
        }

        rt.SetGroup(uri.Hostname(), robots)
        delay := rt.GetDelay(uri, "hopper/different")

        if delay != TestRobotsDelay {
            t.Fatalf("f.GetDelay = %s, want %s", delay, TestRobotsDelay)
        }
    })

}

func TestCrawlable(t *testing.T) {
    f := &Fetcher{}
    f.Init()
    f.Headers.Set("User-Agent", TestUserAgent)

    rt := RobotsTxtMiddleware{Client: f}

    uri, err := url.Parse("http://mock.com/category/resource")
    if err != nil {
        t.Fail()
    }

    t.Run("WithoutGroup", func(t *testing.T) {
        crawlable := rt.Crawlable(uri, "")

        if !crawlable {
            t.Fatalf("f.Crawlable = %t, want %t", crawlable, true)
        }
    })

    t.Run("WithoutGroupPathDisallowed", func(t *testing.T) {
        robots, err := robotstxt.FromString(`
            User-agent: hopper/test

            Sitemap: https://www.example.com/sitemap.xml
        `)
        if err != nil {
            t.Fail()
        }
        
        rt.SetGroup(uri.Hostname(), robots)
        crawlable := rt.Crawlable(uri, "")

        if !crawlable {
            t.Fatalf("f.Crawlable = %t, want %t", crawlable, true)
        }
    })

    t.Run("WithGroupPathDisallowed", func(t *testing.T) {
        robots, err := robotstxt.FromString(`
            User-agent: hopper/test
            Disallow: /category/

            Sitemap: https://www.example.com/sitemap.xml
        `)
        if err != nil {
            t.Fail()
        }
        
        rt.SetGroup(uri.Hostname(), robots)
        crawlable := rt.Crawlable(uri, "")

        if crawlable {
            t.Fatalf("f.Crawlable = %t, want %t", crawlable, false)
        }
    })

    t.Run("WithGroupPathDisallowedOnDifferentAgent", func(t *testing.T) {
        robots, err := robotstxt.FromString(`
            User-agent: hopper/test
            Disallow: /category/

            User-agent: hopper/different
            Disallow: /w/

            Sitemap: https://www.example.com/sitemap.xml
        `)
        if err != nil {
            t.Fail()
        }
        
        rt.SetGroup(uri.Hostname(), robots)
        crawlable := rt.Crawlable(uri, "")

        if crawlable {
            t.Fatalf("f.Crawlable = %t, want %t", crawlable, false)
        }
    })
} 
