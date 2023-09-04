package hopper

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type Response struct {
    StatusCode int
    Body io.ReadCloser

    Request *Request

    Headers http.Header
    Properties map[string]any
}

func NewResponse(r *http.Response, prop map[string]any, req *Request) (*Response, error) {
    res := &Response{
        StatusCode: r.StatusCode,
        Body: r.Body,
        Request: req,
        Headers: r.Header,
        Properties: prop,
    }

    if !res.Valid() {
        return nil, errors.New("Invalid response")
    }

    return res, nil
}

func (res *Response) Do() ([]*Request, error) {
    discovered := []*Request{}

    reader := io.LimitReader(res.Body, res.Properties["ContentLength"].(int64))
    node, err := html.Parse(reader)
    if err != nil {
        return discovered, err
    }

	var f func(*html.Node)

	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
                if attr.Key == "rel" && strings.Contains(attr.Val, "nofollow") {
                    continue
                }
				if attr.Key == "href" {
					resolved, err := res.Request.New("GET", attr.Val)
					if err != nil {
						continue
					}
					discovered = append(discovered, resolved)
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			f(ch)
		}
	}
	f(node)

    return discovered, nil
}

func (res *Response) Valid() bool {
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return false 
	}

	return true
}
