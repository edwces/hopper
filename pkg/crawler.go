package crawler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type Crawler struct {
	// Staritng URLS to crawl
	Seeds []string

	// Used for filtering URLS
	AllowedDomains    []string
	DisallowedDomains []string

	frontier *SafePQueue
	pushChan chan int
	storage  map[string]html.Node
	seenUrls map[string]bool

	mut sync.RWMutex
	wg  sync.WaitGroup

	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
}

// Crawl returns websites data accumulated by crawling over webpages
func (c *Crawler) Crawl() map[string]html.Node {
	c.infoLogger = log.New(os.Stdout, "[INFO]: ", log.LstdFlags)
	c.warningLogger = log.New(os.Stdout, "[WARN]: ", log.LstdFlags)
	c.errorLogger = log.New(os.Stdout, "[ERROR]: ", log.LstdFlags)

	if len(c.Seeds) == 0 {
		c.errorLogger.Fatalln("Seeds to crawl have not been specified")
	}
	if c.AllowedDomains == nil {
		c.AllowedDomains = []string{"*"}
	}
	if c.DisallowedDomains == nil {
		c.DisallowedDomains = []string{""}
	}

	// TODO: maybe make concurrency safe queue with locks
	c.frontier = &SafePQueue{}
	c.pushChan = make(chan int, 100)
	c.storage = map[string]html.Node{}
	c.seenUrls = map[string]bool{}

	c.wg = sync.WaitGroup{}
	c.frontier.Init()

	for _, seed := range c.Seeds {
		c.seenUrls[seed] = true
		c.wg.Add(1)
		c.frontier.Push(&Item{value: seed, priority: 1})
		go func() {
			c.pushChan <- 1
		}()
	}

	// when all goroutines finished close the channel
	go func() {
		c.wg.Wait()
		close(c.pushChan)
	}()

	for range c.pushChan {
		go c.pipe(c.frontier.Pop().value.(string))
	}
	return c.storage
}

func (c *Crawler) pipe(rawUrl string) error {
	defer c.wg.Done()
	c.infoLogger.Printf("Fetching url: %s", rawUrl)

	resp, err := http.Get(rawUrl)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		c.warningLogger.Printf("Could not return response for url: %s", rawUrl)
		return err
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.warningLogger.Printf("Could not read body for url: %s", rawUrl)
		return err
	}

	doc, err := html.Parse(strings.NewReader(string(bytes)))
	if err != nil {
		c.warningLogger.Printf("Could not parse body for url: %s", rawUrl)
		return err
	}

	c.mut.Lock()
	c.storage[rawUrl] = *doc
	c.mut.Unlock()

	extractedUrls := c.extractUrls(doc, rawUrl)
	filteredUrls := filterUrls(extractedUrls, c.AllowedDomains, c.DisallowedDomains)
	dedupedUrls := dedupUrls(filteredUrls)
	unseenUrls := getUnseenUrls(dedupedUrls, c.seenUrls)

	for _, url := range unseenUrls {
		c.wg.Add(1)
		c.frontier.Push(&Item{value: url, priority: 1})
		c.pushChan <- 1

		c.mut.Lock()
		c.seenUrls[url] = true
		c.mut.Unlock()
	}

	return nil
}

func (c *Crawler) extractUrls(node *html.Node, rawUrl string) []string {
	extractedUrls := []string{}

	var f func(node *html.Node, rawUrl string)
	f = func(node *html.Node, rawUrl string) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attribute := range node.Attr {
				if attribute.Key == "href" {
					normalizedUrl, err := normalizeUrl(attribute.Val, rawUrl)
					if err != nil {
						c.warningLogger.Printf("Could not normalize url: %s", err)
						break
					}
					extractedUrls = append(extractedUrls, normalizedUrl)
				}
			}
		}
		for curr := node.FirstChild; curr != nil; curr = curr.NextSibling {
			f(curr, rawUrl)
		}
	}
	f(node, rawUrl)

	return extractedUrls
}

// contains returns true only if passed slice contains passed item
func contains[T comparable](slice []T, itemToCheck T) bool {
	for _, item := range slice {
		if item == itemToCheck {
			return true
		}
	}
	return false
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

// TODO: Optimize deduping algorithm
//
// dedupUrls returns a slice where every url is unique.
func dedupUrls(urls []string) []string {
	dedupedUrls := []string{}
	for _, urlProccessed := range urls {
		if !contains(dedupedUrls, urlProccessed) {
			dedupedUrls = append(dedupedUrls, urlProccessed)
		}
	}
	return dedupedUrls
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
