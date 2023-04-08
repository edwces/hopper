package crawler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func Crawl(urls, allowedDomains, disallowedDomains []string) []html.Node {
	frontier := urls
	// TODO should store pages in some database
	storage := []html.Node{}
	seenUrls := []string{urls[0]}

	for {
		// Fetch/Download data
		urlProccessed := frontier[0]
		// TODO: download robots.txt for domain if not cached

		resp, err := http.Get(urlProccessed)

		if err != nil && resp != nil {
			resp.Body.Close()
			continue
		} else if err != nil {
			continue
		}

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			continue
		}
		utfresp := string(bytes)

		// Parsing process
		doc, err := html.Parse(strings.NewReader(utfresp))
		resp.Body.Close()
		if err != nil {
			continue
		}

		// Callback or Event
		// add data to storage
		storage = append(storage, *doc)

		// -------------- ONLY IF USED AS CRAWLER ------------------
		// extract links
		urlsToAppend := []string{}
		// check if given node is a link and recursively call all off its children
		var f func(*html.Node)
		f = func(node *html.Node) {
			if node.Type == html.ElementNode && node.Data == "a" {
				for _, attribute := range node.Attr {
					if attribute.Key == "href" {
						normalizedUrl, err := normalizeUrl(attribute.Val, urlProccessed)
						if err != nil {
							break
						}
						urlsToAppend = append(urlsToAppend, normalizedUrl)
					}
				}
			}
			for curr := node.FirstChild; curr != nil; curr = curr.NextSibling {
				f(curr)
			}
		}
		f(doc)

		// TODO: filter URL
		filteredUrls := filterUrls(urlsToAppend, allowedDomains, disallowedDomains)
		// TODO: Refactor to add contains func to slice
		// Dedup urlsToAppend
		dedupedUrls := dedupUrls(filteredUrls)
		// check if already has been visited
		unseenUrls := getUnseenUrls(dedupedUrls, seenUrls)
		seenUrls = append(seenUrls, unseenUrls...)
		// --------- END OF CRAWLER SECTION -----------

		// Append urls
		frontier = append(frontier[1:], unseenUrls...)
		if len(frontier) == 0 && len(unseenUrls) == 0 {
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
func getUnseenUrls(urls, seenUrls []string) []string {
	unseenUrls := []string{}
	for _, urlProccessed := range urls {
		if !contains(seenUrls, urlProccessed) {
			unseenUrls = append(unseenUrls, urlProccessed)
		}
	}
	return unseenUrls
}

// TODO: Maybe split this function into traverseNode which runs fun for each node
// and extractUrl which will be the function passed

// TODO: Add better error handling, and maube use lib for url normalization
//
// normalizeUrl returns normalized version of the urlProccessed passed.
func normalizeUrl(urlProccessed string, absoluteUrl string) (string, error) {
	if urlProccessed == "#" || urlProccessed == "javascript:void(0)" {
		return "", errors.New("invalid url")
	}

	if strings.HasPrefix(urlProccessed, "/") {
		parsedAbsoluteUrl, errAbs := url.Parse(absoluteUrl)
		parsedRelativeUrl, errRel := url.Parse(urlProccessed)
		if errAbs != nil || errRel != nil {
			return "", errRel
		}
		return parsedAbsoluteUrl.ResolveReference(parsedRelativeUrl).String(), nil
	}

	if strings.HasPrefix(urlProccessed, "?") {
		parsedAbsoluteUrl, errAbs := url.Parse(absoluteUrl)
		parsedQueryUrl, errQuery := url.Parse(urlProccessed)
		if errAbs != nil || errQuery != nil {
			return "", errQuery
		}
		return parsedAbsoluteUrl.ResolveReference(parsedQueryUrl).String(), nil
	}

	return urlProccessed, nil
}
