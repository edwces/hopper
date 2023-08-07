package hopper

import (
	"math"
	"net/url"
	"sync"
)

type URLQueue struct {
	sync.Mutex

	Free chan int

	threads int
	max     int
	queue   []*url.URL
	seen    map[string]bool
}

func NewURLQueue(max int) *URLQueue {
	return &URLQueue{
		queue: []*url.URL{},
		seen:  map[string]bool{},
		Free:  make(chan int),
		max:   max,
	}
}

func (u *URLQueue) Push(uri *url.URL) {
	u.Lock()
	defer u.Unlock()

	if !u.seen[uri.String()] {
		u.seen[uri.String()] = true
		u.queue = append(u.queue, uri)
	}
	// Send signal to create x new Threads
	// if there's extra items not being proccessed
	// concurrently and if we have free Threads
	balance := len(u.queue) - u.threads
	if balance > 0 && u.threads < u.max {
		u.Free <- int(math.Min(float64(balance), float64(u.max-u.threads)))
		// NOTE: Maybe wait here for all threads to spawn
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

func (u *URLQueue) AddThread() {
	u.Lock()
	defer u.Unlock()

	u.threads++
}

func (u *URLQueue) RemoveThread() {
	u.Lock()
	defer u.Unlock()

	u.threads--

	if u.threads == 0 {
		close(u.Free)
	}
}

func (u *URLQueue) getFreeThreads() int {
	return u.max - u.threads
}
