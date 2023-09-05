package hopper

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultFetcherTimeout = 10 * time.Second
)

type Fetcher struct {
	Client  *http.Client
	Headers http.Header
}

func (f *Fetcher) Init() {
	if f.Client == nil {
		f.Client = http.DefaultClient
		f.Client.Timeout = DefaultFetcherTimeout
	}

	f.Headers = http.Header{}
}

func (f *Fetcher) Do(method string, uri *url.URL, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		return nil, err
	}

    for k, v := range f.Headers {
		if headers.Get(k) == "" {
			headers[k] = v
		}
	}

	req.Header = headers

	return f.Client.Do(req)
}
