package crawler

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	crawler "github.com/crawler/pkg"
)

func TestCrawl(t *testing.T) {
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
							</body>
						</html>`))
	})

	server := httptest.NewUnstartedServer(mux)
	l, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}
	server.Listener = l
	server.Start()
	defer server.Close()

	results := crawler.Crawl([]string{"http://127.0.0.1:8080/"}, []string{"*"}, []string{})
	if len(results) != 3 {
		t.Errorf("Not enough results of a crawl")
	}
}
