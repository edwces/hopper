package crawler

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func NewMockServer() (*httptest.Server, error) {
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
							</body>
						</html>`))
	})

	server := httptest.NewUnstartedServer(mux)
	l, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		return server, err
	}
	server.Listener = l
	return server, nil
}

func TestCrawl(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatalf("Server could not be started")
	}
	server.Start()
	defer server.Close()

	results := Crawl([]string{"http://127.0.0.1:8080/"}, []string{"*"}, []string{})
	if len(results) != 3 {
		t.Errorf("Incorrect length of results: got %d, expected: %d", len(results), 3)
	}
}

func BenchmarkCrawl(b *testing.B) {
	server, err := NewMockServer()
	if err != nil {
		b.Fatalf("Server could not be started")
	}
	server.Start()
	defer server.Close()

	for i := 0; i < b.N; i++ {
		Crawl([]string{"http://127.0.0.1:8080/"}, []string{"*"}, []string{})
	}
}
