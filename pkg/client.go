package hopper

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultClientTimeout = 10 * time.Second
)

type Client struct {
	Client  *http.Client
	Headers http.Header
}

func (c *Client) Init() {
	if c.Client == nil {
		c.Client = http.DefaultClient
		c.Client.Timeout = DefaultClientTimeout
	}

	c.Headers = http.Header{}
}

func (c *Client) Do(method string, uri *url.URL, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		return nil, err
	}

    for k, v := range c.Headers {
		if headers.Get(k) == "" {
			headers[k] = v
		}
	}

	req.Header = headers

	return c.Client.Do(req)
}
