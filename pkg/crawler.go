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
	"sync"
	"time"

	"golang.org/x/net/html"
)

var (
	infoLogger    = log.New(os.Stdout, "[INFO]: ", log.LstdFlags)
	warningLogger = log.New(os.Stdout, "[WARN]: ", log.LstdFlags)
	errorLogger   = log.New(os.Stdout, "[ERROR]: ", log.LstdFlags)
)

type Crawler struct {
	AllowedDomains    []string
	DisallowedDomains []string
	Delay             time.Duration
	Mediatype         string

	frontier *SafePQueue
	storage  map[string]any
	seenUrls map[string]bool

	mut    sync.RWMutex
	wg     sync.WaitGroup
	ticker *time.Ticker
}

// Init initializes default values for crawler.
func (c *Crawler) Init() error {
	if c.Mediatype == "" {
		c.Mediatype = "text/html"
	}
	if c.AllowedDomains == nil {
		c.AllowedDomains = []string{"*"}
	}
	if c.DisallowedDomains == nil {
		c.DisallowedDomains = []string{""}
	}
	if c.Delay == 0 {
		c.Delay = time.Second
	}

	_, _, err := mime.ParseMediaType(c.Mediatype)
	if err != nil {
		errorLogger.Printf("Invalid mime type")
		return err
	}

	c.frontier = &SafePQueue{}
	c.frontier.Init()
	c.storage = map[string]any{}
	c.seenUrls = map[string]bool{}
	c.ticker = time.NewTicker(c.Delay)
	c.wg = sync.WaitGroup{}

	infoLogger.Printf("Crawler initialized succesfully")
	return nil
}

// Crawl returns websites data accumulated by crawling over webpages
func (c *Crawler) Crawl(seeds ...string) map[string]any {

	for _, seed := range seeds {
		c.seenUrls[seed] = true
		c.wg.Add(1)
		c.frontier.Push(&Item{value: seed, priority: 1})
	}

	go func() {
		c.wg.Wait()
		c.frontier.Done()
	}()

	// TODO: should probably be done with channels somehow which are linked to frontier
	for {
		if c.frontier.IsDone() {
			break
		}
		if c.frontier.Len() != 0 {
			go c.Visit(c.frontier.Pop().value.(string))
		}
	}
	return c.storage
}

func (c *Crawler) Visit(uri string) error {
	defer c.wg.Done()

	<-c.ticker.C
	infoLogger.Printf("Fetching url: %s", uri)
	resp, err := http.Get(uri)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		warningLogger.Printf("Could not return response for url: %s", uri)
		return err
	}
	mediatype, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediatype != c.Mediatype && mediatype != "text/html" {
		return nil
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		warningLogger.Printf("Could not read body for url: %s", uri)
		return err
	}

	if mediatype == c.Mediatype {
		c.mut.Lock()
		c.storage[uri] = string(bytes)
		c.mut.Unlock()
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

	for _, url := range unseenUrls {
		c.wg.Add(1)
		c.frontier.Push(&Item{value: url, priority: 1})

		c.mut.Lock()
		c.seenUrls[url] = true
		c.mut.Unlock()
	}

	return nil
}

// extractUrls returns all urls extracted from given node tree
func extractUrls(node *html.Node, uri string) []string {
	extractedUrls := []string{}
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
func filterUrls(urls, allowedDomains, disallowedDomains []string) []string {
	filteredUrls := []string{}
	for _, urlProccessed := range urls {
		parsedUrl, err := url.Parse(urlProccessed)
		if err != nil {
			continue
		}
		if !contains(allowedDomains, "*") && !contains(allowedDomains, parsedUrl.Host) {
			continue
		}
		if contains(disallowedDomains, parsedUrl.Host) {
			continue
		}
		filteredUrls = append(filteredUrls, urlProccessed)
	}
	return filteredUrls
}

// getUnseenUrls returns a set like diferrence
// between first and second passed slices of urls.
func getUnseenUrls(urls []string, seenUrls map[string]bool) []string {
	unseenUrls := []string{}
	for _, urlProccessed := range urls {
		_, doesExist := seenUrls[urlProccessed]
		if !doesExist {
			unseenUrls = append(unseenUrls, urlProccessed)
		}
	}
	return unseenUrls
}

// normalizeUrl returns normalized version of the urlProccessed passed.
func normalizeUrl(rawUrl string, rawBaseUrl string) (string, error) {
	baseUrl, err := url.Parse(rawBaseUrl)
	if err != nil {
		return "", err
	}
	refUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	normalizedUrl := baseUrl.ResolveReference(refUrl)
	if normalizedUrl.Scheme != "http" && normalizedUrl.Scheme != "https" {
		return "", errors.New("unsupported protocol")
	}
	return normalizedUrl.String(), nil
}
