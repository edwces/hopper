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

// Pop sleeps until it can returns url from HostQueue.
func (h *HostQueue) Pop() *url.URL {
	h.mu.Lock()
	defer h.mu.Unlock()

	time.Sleep(time.Until(h.LastVisit.Add(h.Delay)))
	h.LastVisit = time.Now()

	uri := h.queue[len(h.queue)-1]
	h.queue = h.queue[:len(h.queue)-1]

	return uri
}

// Push adds new url to HostQueue.
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

// Pushes requests to HostQueues and creates them if necessary.
// It also sends to channel when it should create new threads.
func (u *RequestQueue) Push(req *Request) {
	u.Lock()
	defer u.Unlock()

	if !u.seen[req.URI.String()] {
        // NOTE HOST items needs to exist all the time
		item, exists := u.hostmap[req.URI.Hostname()]
		if !exists {
			host := NewHostQueue(req.Delay)
			item = &PQueueItem{value: host, priority: int(host.LastVisit.Unix())}
			u.hostmap[req.URI.Hostname()] = item
			heap.Push(u.queue, item)
		} else if item.value.(*HostQueue).Len() == 0 {
			// When we remove heap item we still store it in map for later use
			heap.Push(u.queue, item)
		}

		u.seen[req.URI.String()] = true
		item.value.(*HostQueue).Push(req.URI)
	}
	balance := u.queue.Len() - u.threads
	if balance > 0 && u.threads < u.max {
		u.Free <- int(math.Min(float64(balance), float64(u.max-u.threads)))
	}
}

// Pop returns url from most prioritorized HostQueue and updates heap tree.
func (u *RequestQueue) Pop() *url.URL {
	u.Lock()
	defer u.Unlock()

	item := u.queue.Peek().(*PQueueItem)
	uri := item.value.(*HostQueue).Pop()

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
