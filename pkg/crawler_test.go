package hopper

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const DefaultTestDelay = time.Millisecond * 100

func NewMockServer(port int) (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/main", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
							<head>
								<title>Test Page</title>
							</head>
							<body>
								<a href="/link1"></a>
								<h1>hsdhdjhshjdh</h1>
								<a href="/link2"></a>
								<a href="#"></a>
								<a href="javascript:void(0)"></a>
							</body>
						</html>`))
	})
	mux.HandleFunc("/mime", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
							<head>
								<title>Test Page</title>
							</head>
							<body>
								<h1>hsdhdjhshjdh</h1>
								<a href="/link2"></a>
							</body>
						</html>`))
	})
	mux.HandleFunc("/link1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
							<head>
								<title>Link1 Page</title>
							</head>
							<body>
								<h1>hsdhdjhshjdh</h1>
								<a href="/link2"></a>
							</body>
						</html>`))
	})
	mux.HandleFunc("/link2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
							<head>
								<title>Link2 Page</title>
							</head>
							<body>
								<h1>hsdhdjhshjdh</h1>
								<a href="/link1"></a>
								<a href="/link3"></a>
							</body>
						</html>`))
	})
	mux.HandleFunc("/link3", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Hello Mars")
	})
	mux.HandleFunc("/cross", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
							<head>
								<title>Link2 Page</title>
							</head>
							<body>
								<h1>hsdhdjhshjdh</h1>
								<a href="http://127.0.0.1:8081/link1"></a>
								<a href="http://127.0.0.1:8081/link2"></a>
								<a href="/link1"></a>
								<a href="/link2"></a>
							</body>
						</html>`))
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:8080/link1", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(`User-Agent: *
						Disallow: /excluded1
						Disallow: /excluded2
		`))
	})
	mux.HandleFunc("/excluded1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
							<head>
								<title>Secret Page1</title>
							</head>
							<body>
								<h1>hsdhdjhshjdh</h1>
							</body>
						</html>`))
	})
	mux.HandleFunc("/excluded2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
							<head>
								<title>Secret Page2</title>
							</head>
							<body>
								<h1>hsdhdjhshjdh</h1>
							</body>
						</html>`))
	})
	mux.HandleFunc("/robot", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
							<head>
								<title>Secret Page2</title>
							</head>
							<body>
								<h1>hsdhdjhshjdh</h1>
								<a href="/excluded1"></a>
								<a href="/link1"></a>
								<a href="/excluded2"></a>
							</body>
						</html>`))
	})

	server := httptest.NewUnstartedServer(mux)
	host := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := net.Listen("tcp", host)
	if err != nil {
		return server, err
	}
	server.Listener = l
	return server, nil
}

func TestCrawl(t *testing.T) {
	server, err := NewMockServer(8080)
	if err != nil {
		t.Fatalf("Server could not be started")
	}
	server.Start()
	defer server.Close()

	crawler := Crawler{Delay: DefaultTestDelay}
	crawler.Init()
	results := crawler.Crawl("http://127.0.0.1:8080/main")
	if len(results) != 3 {
		t.Errorf("Incorrect length of results: got %d, expected: %d", len(results), 3)
	}
}

func TestCrawlMime(t *testing.T) {
	server, err := NewMockServer(8080)
	if err != nil {
		t.Fatalf("Server could not be started")
	}
	server.Start()
	defer server.Close()

	crawler := Crawler{Mediatype: "text/plain", Delay: DefaultTestDelay}
	crawler.Init()
	results := crawler.Crawl("http://127.0.0.1:8080/mime")
	if len(results) != 1 {
		t.Errorf("Incorrect length of results: got %d, expected: %d", len(results), 1)
	}

}

func TestCrawlCross(t *testing.T) {
	server1, err := NewMockServer(8080)
	if err != nil {
		t.Fatalf("Server could not be started")
	}
	server2, err := NewMockServer(8081)
	if err != nil {
		t.Fatalf("Server could not be started")
	}

	server1.Start()
	defer server1.Close()
	server2.Start()
	defer server2.Close()

	crawler := Crawler{Delay: DefaultTestDelay}
	crawler.Init()
	results := crawler.Crawl("http://127.0.0.1:8080/cross")

	if len(results) != 5 {
		t.Errorf("Incorrect length of results: got %d, expected: %d", len(results), 5)
	}
}

func TestCrawlRobotsTxt(t *testing.T) {
	server1, err := NewMockServer(8080)
	if err != nil {
		t.Fatalf("Server could not be started")
	}
	server1.Start()
	defer server1.Close()

	crawler := Crawler{Delay: DefaultTestDelay}
	crawler.Init()
	results := crawler.Crawl("http://127.0.0.1:8080/robot")
	if len(results) != 3 {
		t.Errorf("Incorrect length of results: got %d, expected: %d", len(results), 3)
	}
}

func TestCrawlRedirect(t *testing.T) {
	server1, err := NewMockServer(8080)
	if err != nil {
		t.Fatalf("Server could not be started")
	}
	server1.Start()
	defer server1.Close()

	crawler := Crawler{Delay: DefaultTestDelay}
	crawler.Init()
	results := crawler.Crawl("http://127.0.0.1:8080/redirect")

	if len(results) != 2 {
		t.Errorf("Incorrect length of results: got %d, expected: %d", len(results), 2)
	}
}

func TestCrawlRequestUserAgent(t *testing.T) {
	crawler := Crawler{Delay: DefaultTestDelay}
	crawler.Init()
	req := crawler.newRequest("GET", "http://127.0.0.1:8080/main")
	userAgent := req.Header.Get("User-Agent")
	if userAgent != DefaultUserAgent {
		t.Errorf("Incorrect user-agent header set: got %s, expected %s", userAgent, DefaultUserAgent)
	}
}
