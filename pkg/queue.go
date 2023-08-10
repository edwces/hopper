package hopper

import (
	"container/heap"
	"math"
	"net/url"
	"sync"
	"time"
)

type Request struct {
	URI *url.URL

	Delay time.Duration
}

type PQueueItem struct {
	value    any
	priority int
	index    int
}

type PQueue []*PQueueItem

func (pq PQueue) Len() int { return len(pq) }

func (pq PQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

func (pq PQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PQueue) Push(x any) {
	n := len(*pq)
	item := x.(*PQueueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

func (pq *PQueue) Update(item *PQueueItem, value any, priority int) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}

func (pq *PQueue) Peek() any {
	q := *pq
	n := len(q)
	return q[n-1]
}

type HostQueue struct {
	mu sync.Mutex

	LastVisit time.Time
	Delay     time.Duration
	queue     []*url.URL
}

func NewHostQueue(delay time.Duration) *HostQueue {
	return &HostQueue{
		LastVisit: time.Now().Add(-delay),
		Delay:     delay,
		queue:     []*url.URL{},
		mu:        sync.Mutex{},
	}
}

func (h *HostQueue) Pop() *url.URL {
	h.mu.Lock()
	defer h.mu.Unlock()

	time.Sleep(time.Until(h.LastVisit.Add(h.Delay)))
	h.LastVisit = time.Now()

	uri := h.queue[len(h.queue)-1]
	h.queue = h.queue[:len(h.queue)-1]

	return uri
}

func (h *HostQueue) Push(uri *url.URL) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.queue = append(h.queue, uri)
}

func (h *HostQueue) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.queue)
}

type RequestQueue struct {
	sync.Mutex

	Free chan int

	threads int
	max     int
	queue   *PQueue
	hostmap map[string]*PQueueItem
	seen    map[string]bool
}

func NewRequestQueue(max int) *RequestQueue {
	q := &RequestQueue{
		queue:   &PQueue{},
		hostmap: map[string]*PQueueItem{},
		seen:    map[string]bool{},
		Free:    make(chan int),
		max:     max,
	}
	heap.Init(q.queue)
	return q
}

func (u *RequestQueue) Push(req *Request) {
	u.Lock()
	defer u.Unlock()

	if !u.seen[req.URI.String()] {
		item, exists := u.hostmap[req.URI.Hostname()]
		if !exists {
			// NOTE HOST items needs to exist all the time
			host := NewHostQueue(req.Delay)
			item = &PQueueItem{value: host, priority: int(host.LastVisit.Unix())}
			u.hostmap[req.URI.Hostname()] = item
			heap.Push(u.queue, item)
			// When we remove heap item we still store it in map for later use
		} else if item.value.(*HostQueue).Len() == 0 {
			heap.Push(u.queue, item)
		}

		u.seen[req.URI.String()] = true
		item.value.(*HostQueue).Push(req.URI)
	}
	// Send signal to create x new Threads
	// if there's extra items not being proccessed
	// concurrently and if we have free Threads
	balance := u.queue.Len() - u.threads
	if balance > 0 && u.threads < u.max {
		u.Free <- int(math.Min(float64(balance), float64(u.max-u.threads)))
		// NOTE: Maybe wait here for all threads to spawn
	}
}

func (u *RequestQueue) Pop() *url.URL {
	u.Lock()
	defer u.Unlock()

	// Get most prioritozed host
	item := u.queue.Peek().(*PQueueItem)
	// Get it's uri
	uri := item.value.(*HostQueue).Pop()
	// if host queue is empty delete it from heap else update heap
	if item.value.(*HostQueue).Len() == 0 {
		heap.Remove(u.queue, item.index)
	} else {
		u.queue.Update(item, item.value, int(item.value.(*HostQueue).LastVisit.Unix()))
	}

	return uri
}

func (u *RequestQueue) Len() int {
	u.Lock()
	defer u.Unlock()

	return u.queue.Len()
}

func (u *RequestQueue) AddThread() {
	u.Lock()
	defer u.Unlock()

	u.threads++
}

func (u *RequestQueue) RemoveThread() {
	u.Lock()
	defer u.Unlock()

	u.threads--

	if u.threads == 0 {
		close(u.Free)
	}
}
