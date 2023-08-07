package hopper

import (
	"net/url"
	"sync"
)

type URLQueue struct {
    sync.Mutex
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
    u.Lock()
    defer u.Unlock()

	if !u.seen[uri.String()] {
		u.seen[uri.String()] = true
		u.queue = append(u.queue, uri)
	}
}

func (u *URLQueue) Pop() *url.URL {
    u.Lock()
    defer u.Unlock()

	uri := u.queue[len(u.queue)-1]
	u.queue = u.queue[:len(u.queue)-1]
	return uri
}

func (u *URLQueue) Length() int {
    u.Lock()
    defer u.Unlock()

	return len(u.queue)
}
