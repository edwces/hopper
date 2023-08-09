package hopper

import (
	"container/heap"
	"net/url"
	"sync"
	"time"
)

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

type Host struct {
    mu sync.Mutex

    LastVisit time.Time
    Delay time.Duration
    queue []*url.URL
}

func NewHost(delay time.Duration) *Host {
    return &Host{
        LastVisit: time.Now().Add(-delay),
        Delay: delay,
        queue: []*url.URL{},
        mu: sync.Mutex{},
    }
}

func (h *Host) Pop() *url.URL {
    h.mu.Lock()
    defer h.mu.Unlock()

    time.Sleep(time.Until(h.LastVisit.Add(h.Delay)))
    h.LastVisit = time.Now()

    uri := h.queue[len(h.queue)-1]
    h.queue = h.queue[:len(h.queue)-1]

    return uri
}

func (h *Host) Push(uri *url.URL) {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.queue = append(h.queue, uri)
}

func (h *Host) Len() int {
    h.mu.Lock()
    defer h.mu.Unlock()

    return len(h.queue)
}
