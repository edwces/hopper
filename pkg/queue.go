package hopper

import "net/url"

type URLQueue struct {
	queue []*url.URL
	seen  map[string]bool
}

func NewURLQueue() *URLQueue {
	return &URLQueue{
		queue: []*url.URL{},
		seen:  map[string]bool{},
	}
}

func (u *URLQueue) Push(uri *url.URL) {
	if !u.seen[uri.String()] {
		u.seen[uri.String()] = true
		u.queue = append(u.queue, uri)
	}
}

func (u *URLQueue) Pop() *url.URL {
	uri := u.queue[len(u.queue)-1]
	u.queue = u.queue[:len(u.queue)-1]
	return uri
}

func (u *URLQueue) Length() int {
	return len(u.queue)
}
