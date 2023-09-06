package hopper

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultUserAgent = "hopper/0.1"
	DefaultDelay     = time.Second * 15
)

type Request struct {
	Method string
	URL    *url.URL
	Depth  int

	Headers    http.Header
	Properties map[string]any
}

func (req *Request) Init() {
	req.URL = &url.URL{}
	req.Headers = http.Header{} 
	req.Properties = map[string]any{}
	req.Depth = -1
}

func (req Request) New(method string, uri string) (*Request, error) {
	parsed, err := req.URL.Parse(uri)
	if err != nil {
		return nil, err
        
	}
	if !parsed.IsAbs() {
		return nil, errors.New("Relative url can't be resolved")
	}
	parsed.Fragment = ""

	req.URL = parsed
	req.Method = method
    req.Headers = http.Header{}
	req.Depth++

    // NOTE: Because properties are a map we are deep copying it to new request
    newProperties := map[string]any{}
    for k, v := range req.Properties {
        newProperties[k] = v
    }
    req.Properties = newProperties

	if !req.Valid() {
		return nil, errors.New("Invalid request")
	}

	return &req, nil
}

func (req *Request) Valid() bool {
   	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return false
	}

	if req.Depth > req.Properties["AllowedDepth"].(int) {
		return false
	}

	for _, host := range req.Properties["AllowedDomains"].([]string) {
		if req.URL.Hostname() == host {
			continue
		}
		return false
	}

	for _, host := range req.Properties["DisallowedDomains"].([]string) {
		if req.URL.Hostname() != host {
			continue
		}
		return false
	}

	return true
}
