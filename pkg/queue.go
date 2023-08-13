package hopper

import (
	"container/heap"
	"math"
	"net/url"
	"runtime"
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

type DelayedQueue struct {
	delay time.Duration
    clock time.Time
	queue []*url.URL
}

func NewDelayedQueue(delay time.Duration) *DelayedQueue {
	return &DelayedQueue{
		clock: time.Now().Add(-delay),
		delay: delay,
		queue: []*url.URL{},
	}
}

// Pop sleeps until it can returns url from HostQueue.
func (h *DelayedQueue) Pop() *url.URL {
	time.Sleep(time.Until(h.clock.Add(h.delay)))
	h.clock = time.Now()

	uri := h.queue[len(h.queue)-1]
	h.queue = h.queue[:len(h.queue)-1]

	return uri
}

// Push adds new url to HostQueue.
func (h *DelayedQueue) Push(uri *url.URL) {
	h.queue = append(h.queue, uri)
}

func (h *DelayedQueue) Len() int {
	return len(h.queue)
}

type URLQueue struct {
	sync.Mutex

	Free chan int

	threads int
	max     int
	queue   *PQueue
	itemMap map[string]*PQueueItem
	seen    map[string]bool
}

func (u *URLQueue) Init() {
	if u.max == 0 {
        u.max = runtime.GOMAXPROCS(0)
    }

	u.queue = &PQueue{}
	u.itemMap = map[string]*PQueueItem{}
	u.seen = map[string]bool{}
	u.Free = make(chan int)

	heap.Init(u.queue)
}

// Pushes requests to HostQueues and creates them if necessary.
// It also sends to channel when it should create new threads.
func (u *URLQueue) Push(uri *url.URL, delay time.Duration) {
	u.Lock()
	defer u.Unlock()

	if !u.seen[uri.String()] {
        hqueue := u.getHostQueue(uri, delay)
		hqueue.Push(uri)
		u.seen[uri.String()] = true
	}

	balance := u.queue.Len() - u.threads
	if balance > 0 && u.threads < u.max {
        // BUG: This part is not synchronized correctly
        // It can send multiple messages before workers will increase number of threads
		u.Free <- int(math.Min(float64(balance), float64(u.max-u.threads)))
	}
}

// Pop returns url from most prioritorized HostQueue and updates heap tree.
func (u *URLQueue) Pop() *url.URL {
	u.Lock()
	defer u.Unlock()

	item := u.queue.Peek().(*PQueueItem)
	uri := item.value.(*DelayedQueue).Pop()

	if item.value.(*DelayedQueue).Len() == 0 {
		heap.Remove(u.queue, item.index)
	} else {
		u.queue.Update(item, item.value, int(item.value.(*DelayedQueue).clock.Unix()))
	}

	return uri
}

// getHostQueue returns DelayedQueue for equivalent hostname and creates it,
// if one does not exists.
func (u *URLQueue) getHostQueue(uri *url.URL, delay time.Duration) *DelayedQueue {
    item, exists := u.itemMap[uri.Hostname()]
    if !exists {
        queue := NewDelayedQueue(delay)
        item = &PQueueItem{value: queue, priority: int(queue.clock.Unix())}
        u.itemMap[uri.Hostname()] = item
        heap.Push(u.queue, item)
    } else if item.value.(*DelayedQueue).Len() == 0 {
        // When we remove heap item we still store it in map for later use
        heap.Push(u.queue, item)
    }
    return item.value.(*DelayedQueue)
}

func (u *URLQueue) Len() int {
	u.Lock()
	defer u.Unlock()

	return u.queue.Len()
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
