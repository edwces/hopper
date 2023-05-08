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
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
								<a href="/mime"></a>
							</body>
						</html>`))
	})
	mux.HandleFunc("/mime", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "plain/text")
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
	results := crawler.Crawl("http://127.0.0.1:8080/")
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

	crawler := Crawler{Mediatype: "plain/text", Delay: DefaultTestDelay}
	crawler.Init()
	results := crawler.Crawl("http://127.0.0.1:8080/")
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

func BenchmarkCrawl(b *testing.B) {
	server, err := NewMockServer(8080)
	if err != nil {
		b.Fatalf("Server could not be started")
	}
	server.Start()
	defer server.Close()

	for i := 0; i < b.N; i++ {
		crawler := Crawler{}
		crawler.Init()
		crawler.Crawl("http://127.0.0.1:8080/")
	}
}
