package hopper

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/temoto/robotstxt"
)

const (
	TestRobotsDelay = time.Second
    TestDelay = time.Millisecond * 500
	TestUserAgent   = "hopper/test"
)

func TestRobotsTxt(t *testing.T) {
	robots := `
        User-Agent: hopper/delay
        Crawl-Delay: 1
        
        User-Agent: hopper/disallow
        Disallow: /1
        Disallow: /3
    `
	index := `
        <!DOCTYPE html><html><body>
        <a href="/1"></a>
        <a href="/2"></a>
        <a href="/3"></a>
        </body></html>
    `
	body := `
        <!DOCTYPE html><html><body>
        <h1>My First Heading</h1>
        <p>My first paragraph.</p>
        </body></html>
    `

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/":
			io.WriteString(w, index)
		case "/1":
			io.WriteString(w, body)
		case "/2":
			io.WriteString(w, body)
		case "/3":
			io.WriteString(w, body)
		case "/robots.txt":
			io.WriteString(w, robots)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))

    defer srv.Close()

	t.Run("WithDelay", func(t *testing.T) {
        crawler := Crawler{Client: srv.Client(), UserAgent: "hopper/delay", Delay: TestDelay}
		crawler.Init()
		RobotsTxt(&crawler)

        type Result struct {
            Path string
            Delay time.Duration
            Exists bool
        }

        var results []Result

		crawler.OnPush(func(r *Request) error {
            delay, exists := r.Properties["Delay"].(time.Duration)
            results = append(results, Result{Path: r.URL.Path, Delay: delay, Exists: exists})

			return nil
		})

        crawler.Run(srv.URL)

        for _, result := range results {
            t.Run(result.Path, func(t *testing.T) {
                if result.Path == "" {
                    return
                }

                if !result.Exists {
                    t.Fatalf("Request.Properties[Delay] = nil, want %s", TestRobotsDelay)
                }

                if result.Delay != TestRobotsDelay {
                    t.Fatalf("Request.Properties[Delay] = %s, want %s", result.Delay, TestRobotsDelay)
                }
            })
        }   
	})

    t.Run("WithDisallow", func(t *testing.T) {
        crawler := Crawler{Client: srv.Client(), UserAgent: "hopper/disallow", Delay: TestDelay}
		crawler.Init()
		RobotsTxt(&crawler)

        results := map[string]error{}

        crawler.OnError(func(r *Request, err error) {
            results[r.URL.Path] = err
        })

        crawler.Run(srv.URL)

        for path, err := range results {
            t.Run(path, func(t *testing.T) {
                if err != nil && (path == "/2" || path == "") {
                    t.Fatalf("err = %T, want nil", err)
                } 

                if err == nil && (path == "/1" || path == "/3") {
                    t.Fatalf("err = nil, want error")
                }
            })
        }
	})
}

func TestGetDelay(t *testing.T) {
	f := &Client{}
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
            Crawl-Delay: 1
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
            Crawl-Delay: 1
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
	f := &Client{}
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
