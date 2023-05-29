package hopper

import (
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/temoto/robotstxt"
	"golang.org/x/net/html"
)

var (
	infoLogger    = log.New(os.Stdout, "[INFO]: ", log.LstdFlags)
	warningLogger = log.New(os.Stdout, "[WARN]: ", log.LstdFlags)
	errorLogger   = log.New(os.Stdout, "[ERROR]: ", log.LstdFlags)
)

const (
	DefaultDelay     = time.Second * 10
	DefaulMediatype  = "text/html"
	DefaultUserAgent = "hopper (https://github.com/edwces/hopper)"
)

type Crawler struct {
	AllowedDomains    []string
	DisallowedDomains []string
	Delay             time.Duration
	Mediatype         string
	Client            *http.Client
	UserAgent         string

	queue     *InMemoryURLQueue
	storage   map[string]any
	seenUrls  map[string]bool
	robotsMap map[string]*robotstxt.RobotsData
}

// Init initializes default values for crawler.
func (c *Crawler) Init() error {
	if c.Mediatype == "" {
		c.Mediatype = DefaulMediatype
	}
	if c.AllowedDomains == nil {
		c.AllowedDomains = []string{"*"}
	}
	if c.DisallowedDomains == nil {
		c.DisallowedDomains = []string{""}
	}
	if c.Delay == 0 {
		c.Delay = DefaultDelay
	}
	if c.UserAgent == "" {
		c.UserAgent = DefaultUserAgent
	}
	if c.Client == nil {
		c.Client = http.DefaultClient
		c.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	_, _, err := mime.ParseMediaType(c.Mediatype)
	if err != nil {
		errorLogger.Fatal("Invalid mime type")
		return err
	}

	c.queue = &InMemoryURLQueue{Delay: c.Delay}
	c.queue.Init()
	c.storage = map[string]any{}
	c.seenUrls = map[string]bool{}
	c.robotsMap = map[string]*robotstxt.RobotsData{}

	infoLogger.Printf("Crawler initialized succesfully")
	return nil
}

// Crawl returns websites data accumulated by crawling over webpages
func (c *Crawler) Crawl(seeds ...string) map[string]any {
	for _, seed := range seeds {
		uri, err := url.Parse(seed)
		if err != nil {
			warningLogger.Printf("can't parse seed: %s", seed)
			continue
		}
		c.seenUrls[uri.String()] = true
		c.queue.Push(uri)
	}

	for c.queue.Len() != 0 {
		c.Visit(c.queue.Pop())
	}

	return c.storage
}

func (c *Crawler) newRequest(method, rawUrl string) *http.Request {
	req, err := http.NewRequest(method, rawUrl, nil)
	if err != nil {
		errorLogger.Fatalf("Invalid request: %s %s", method, rawUrl)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	return req
}

func (c *Crawler) handleRedirect(resp *http.Response, uri *url.URL) error {
	infoLogger.Printf("Redirecting from url: %s", uri.String())
	loc, err := resp.Location()
	if err != nil {
		return err
	}
	// handle infinite redirect
	if loc.String() == uri.String() {
		warningLogger.Printf("Infinite redirect from url: %s", uri.String())
		return errors.New("infinite redirect")
	}
	c.queue.Push(loc)
	c.seenUrls[loc.String()] = true
	return nil
}

func (c *Crawler) fetchRobotsTxt(uri url.URL) (*robotstxt.RobotsData, error) {
	robots, exists := c.robotsMap[uri.Host]
	if !exists {
		infoLogger.Printf("Fetching robots.txt for host: %s", uri.Host)
		uri.Path = "/robots.txt"
		req := c.newRequest("GET", uri.String())
		resp, err := c.Client.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return nil, err
		}
		robots, err = robotstxt.FromResponse(resp)
		if err != nil {
			return nil, err
		}
		c.robotsMap[uri.Host] = robots
		agentGroup := robots.FindGroup(c.UserAgent)
		c.queue.Update(uri.Host, agentGroup.CrawlDelay)
	}
	return robots, nil
}

func (c *Crawler) Visit(rawUrl string) error {
	uri, err := url.Parse(rawUrl)
	if err != nil {
		errorLogger.Fatalf("Could not parse uri: %s", uri.String())
	}

	// check robots.txt exclusion
	robots, err := c.fetchRobotsTxt(*uri)
	if err != nil {
		warningLogger.Printf("Could not fetch robots.txt for url: %s", uri.String())
	} else {
		agentGroup := robots.FindGroup(c.UserAgent)
		if !agentGroup.Test(uri.EscapedPath()) {
			warningLogger.Printf("Robots.txt exclusion for url: %s", uri.String())
			return errors.New("robots.txt exclusion")
		}
	}

	// check mimetype before so we don't need to download full body
	infoLogger.Printf("Requesting headers for url: %s", uri.String())
	req := c.newRequest("HEAD", uri.String())
	headResp, err := c.Client.Do(req)
	if headResp != nil {
		defer headResp.Body.Close()
	}
	if err != nil {
		warningLogger.Printf("Could not return response for url: %s", uri.String())
		return err
	}

	mediatype, _, err := mime.ParseMediaType(headResp.Header.Get("Content-Type"))
	if mediatype != c.Mediatype && mediatype != "text/html" || err != nil {
		return nil
	}

	infoLogger.Printf("Fetching url: %s", uri.String())
	req = c.newRequest("GET", uri.String())
	resp, err := c.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		warningLogger.Printf("Could not return response for url: %s", uri)
		return err
	}

	// handle redirects
	if resp.StatusCode > 300 && resp.StatusCode < 400 {
		return c.handleRedirect(resp, uri)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		warningLogger.Printf("Could not read body for url: %s", uri)
		return err
	}

	if mediatype == c.Mediatype {
		c.storage[uri.String()] = string(bytes)
	}
	if mediatype != "text/html" {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(string(bytes)))
	if err != nil {
		warningLogger.Printf("Could not parse body for url: %s", uri)
		return err
	}

	extractedUrls := extractUrls(doc, uri)
	filteredUrls := filterUrls(extractedUrls, c.AllowedDomains, c.DisallowedDomains)
	dedupedUrls := dedup(filteredUrls)
	unseenUrls := getUnseenUrls(dedupedUrls, c.seenUrls)

	for _, unseenUrl := range unseenUrls {
		c.queue.Push(unseenUrl)
		c.seenUrls[rawUrl] = true
	}

	return nil
}

// extractUrls returns all urls extracted from given node tree
func extractUrls(node *html.Node, uri *url.URL) []*url.URL {
	extractedUrls := []*url.URL{}
	searchNode(node, func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attribute := range node.Attr {
				if attribute.Key == "href" {
					normalizedUrl, err := normalizeUrl(attribute.Val, uri)
					if err != nil {
						warningLogger.Printf("Could not normalize url: %s", err)
						break
					}
					extractedUrls = append(extractedUrls, normalizedUrl)
				}
			}
		}
	})
	return extractedUrls
}

// searchNode executes func for each node in a tree.
func searchNode(node *html.Node, fn func(*html.Node)) {
	fn(node)

	for curr := node.FirstChild; curr != nil; curr = curr.NextSibling {
		searchNode(curr, fn)
	}
}

// contains returns true only if passed slice contains passed item.
func contains[T comparable](slice []T, itemToCheck T) bool {
	for _, item := range slice {
		if item == itemToCheck {
			return true
		}
	}
	return false
}

// dedup returns a slice where every element is unique.
func dedup[T comparable](slice []T) []T {
	deduped := []T{}
	dedupedMap := map[T]bool{}
	for _, item := range slice {
		if !dedupedMap[item] {
			deduped = append(deduped, item)
			dedupedMap[item] = true
		}
	}
	return deduped

}

// filterUrls returns a urls filtered based on passed filter rules
func filterUrls(urls []*url.URL, allowedDomains, disallowedDomains []string) []*url.URL {
	filteredUrls := []*url.URL{}
	for _, urlProccessed := range urls {
		if !contains(allowedDomains, "*") && !contains(allowedDomains, urlProccessed.Host) {
			continue
		}
		if contains(disallowedDomains, urlProccessed.Host) {
			continue
		}
		filteredUrls = append(filteredUrls, urlProccessed)
	}
	return filteredUrls
}

// getUnseenUrls returns a set like diferrence
// between first and second passed slices of urls.
func getUnseenUrls(urls []*url.URL, seenUrls map[string]bool) []*url.URL {
	unseenUrls := []*url.URL{}
	for _, urlProccessed := range urls {
		_, doesExist := seenUrls[urlProccessed.String()]
		if !doesExist {
			unseenUrls = append(unseenUrls, urlProccessed)
		}
	}
	return unseenUrls
}

// normalizeUrl returns normalized version of the urlProccessed passed.
func normalizeUrl(ref string, uri *url.URL) (*url.URL, error) {
	refUrl, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}
	normalizedUrl := uri.ResolveReference(refUrl)
	if normalizedUrl.Scheme != "http" && normalizedUrl.Scheme != "https" {
		return nil, errors.New("unsupported protocol")
	}
	return normalizedUrl, nil
}
