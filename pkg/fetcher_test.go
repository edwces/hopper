package hopper

import (
	"net/url"
	"testing"
	"time"

	"github.com/temoto/robotstxt"
)

const (
    TestDelay = 5 * time.Second
    TestRobotsDelay = 10 * time.Second
    TestUserAgent = "hopper/test"
)

func TestFetcherDelay(t *testing.T) {
    f := Fetcher{Delay: TestDelay}
    f.Init()
    f.Headers["User-Agent"] = TestUserAgent

    uri, err := url.Parse("http://mock.com/hello")
    if err != nil {
        t.Fail()
    }
    
    t.Run("group=nil", func(t *testing.T) {
        delay := f.GetDelay(uri)

        if delay != TestDelay {
            t.Fatalf("f.GetDelay = %s, want %s", delay, TestDelay)
        }
    })

    t.Run("group.CrawlDelay=0", func(t *testing.T) {
        robots, err := robotstxt.FromString(`
            User-agent: hopper/test
            Disallow: /w/

            Sitemap: https://www.example.com/sitemap.xml
        `)
        if err != nil {
            t.Fail()
        }

        f.SetRobots(uri.Hostname(), robots)
        delay := f.GetDelay(uri)

        if delay != TestDelay {
            t.Fatalf("f.GetDelay = %s, want %s", delay, TestDelay)
        }
    })

    t.Run("group.CrawlDelay!=0", func(t *testing.T) {
        robots, err := robotstxt.FromString(`
            User-agent: hopper/test
            Crawl-Delay: 10
            Disallow: /w/

            Sitemap: https://www.example.com/sitemap.xml
        `)
        if err != nil {
            t.Fail()
        }

        f.SetRobots(uri.Hostname(), robots)
        delay := f.GetDelay(uri)

        if delay != TestRobotsDelay {
            t.Fatalf("f.GetDelay = %s, want %s", delay, TestRobotsDelay)
        }
    })

}
