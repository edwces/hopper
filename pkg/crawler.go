package crawler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func Crawl(urls, allowedDomains, disallowedDomains []string) map[string]html.Node {
	// TODO: should be a multithreaded queue
	frontier := NewQueue(urls...)
	storage := map[string]html.Node{}
	seenUrls := map[string]bool{}

	infoLogger := log.New(os.Stdout, "[INFO]: ", log.LstdFlags)
	warningLogger := log.New(os.Stdout, "[WARN]: ", log.LstdFlags)

	for {
		rawUrl := frontier.Dequeue()
		infoLogger.Printf("Fetching url: %s", rawUrl)
		// TODO: download robots.txt for domain if not cached
		resp, err := http.Get(rawUrl)

		if err != nil && resp != nil {
			resp.Body.Close()
			warningLogger.Printf("Error fetching url: %s", err)
			continue
		} else if err != nil {
			warningLogger.Printf("Error fetching url: %s", err)
			continue
		}

		bytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			warningLogger.Printf("Error reading response body: %s", err)
			continue
		}
		body := string(bytes)

		// Parsing process
		doc, err := html.Parse(strings.NewReader(body))
		if err != nil {
			warningLogger.Printf("Error parsing response body: %s", err)
			continue
		}

		// Callback or Event
		// add data to storage
		storage[rawUrl] = *doc

		// -------------- ONLY IF USED AS CRAWLER ------------------
		// extract links
		extractedUrls := []string{}
		// check if given node is a link and recursively call all off its children
		var f func(*html.Node)
		f = func(node *html.Node) {
			if node.Type == html.ElementNode && node.Data == "a" {
				for _, attribute := range node.Attr {
					if attribute.Key == "href" {
						normalizedUrl, err := normalizeUrl(attribute.Val, rawUrl)
						if err != nil {
							warningLogger.Printf("Error normalizing url: %s", err)
							break
						}
						extractedUrls = append(extractedUrls, normalizedUrl)
					}
				}
			}
			for curr := node.FirstChild; curr != nil; curr = curr.NextSibling {
				f(curr)
			}
		}
		f(doc)

		// filter URL
		filteredUrls := filterUrls(extractedUrls, allowedDomains, disallowedDomains)
		// Dedup urlsToAppend
		dedupedUrls := dedupUrls(filteredUrls)
		// check if already has been visited
		seenUrls[rawUrl] = true
		unseenUrls := getUnseenUrls(dedupedUrls, seenUrls)
		// --------- END OF CRAWLER SECTION -----------

		// Append urls
		for _, unseenUrl := range unseenUrls {
			frontier.Enqueue(unseenUrl)
			seenUrls[unseenUrl] = true
		}

		if frontier.size == 0 {
			break
		}
	}
	return storage
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
