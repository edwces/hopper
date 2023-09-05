package hopper

import (
	"container/heap"
	"math"
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
	clock time.Time
	queue []*Request
}

// Pop sleeps until it can returns url from HostQueue.
func (h *DelayedQueue) Pop() *Request {

	req := h.queue[len(h.queue)-1]
	h.queue = h.queue[:len(h.queue)-1]
    
    time.Sleep(time.Until(h.clock.Add(req.Properties["Delay"].(time.Duration))))
	h.clock = time.Now()

	return req
}

// Push adds new url to HostQueue.
func (h *DelayedQueue) Push(req *Request) {
	h.queue = append(h.queue, req)
}

func (h *DelayedQueue) Len() int {
	return len(h.queue)
}

type URLQueue struct {
	sync.Mutex

	Free chan int
	Max  int

	threads int
	queue   *PQueue
	itemMap map[string]*PQueueItem
	seen    map[string]bool
}

func (u *URLQueue) Init() {
	if u.Max == 0 {
		u.Max = runtime.GOMAXPROCS(0)
	}
	u.queue = &PQueue{}
	u.itemMap = map[string]*PQueueItem{}
	u.seen = map[string]bool{}
	u.Free = make(chan int)

	heap.Init(u.queue)
}

// Pushes requests to HostQueues and creates them if necessary.
// It also sends to channel when it should create new threads.
func (u *URLQueue) Push(req *Request) {
	u.Lock()
	defer u.Unlock()

	if !u.seen[req.URL.String()] {
		hq := u.getHostQueue(req)
		hq.Push(req)
		u.seen[req.URL.String()] = true
	}

	balance := u.queue.Len() - u.threads
	if balance > 0 && u.threads < u.Max {
		free := int(math.Min(float64(balance), float64(u.Max-u.threads)))
		u.threads += free
		u.Free <- free
	}
}

// Pop returns url from most prioritorized HostQueue and updates heap tree.
func (u *URLQueue) Pop() *Request {
	u.Lock()
	defer u.Unlock()

	if u.threads == 0 {
		close(u.Free)
	}

	item := u.queue.Peek().(*PQueueItem)
	req := item.value.(*DelayedQueue).Pop()

	if item.value.(*DelayedQueue).Len() == 0 {
		heap.Remove(u.queue, item.index)
	} else {
		u.queue.Update(item, item.value, int(item.value.(*DelayedQueue).clock.Unix()))
	}

	balance := u.queue.Len() - u.threads
	if balance < 0 && u.threads <= u.Max {
		u.threads += balance
	}

	return req
}

// getHostQueue returns DelayedQueue for equivalent hostname and creates it,
// if one does not exists.
func (u *URLQueue) getHostQueue(req *Request) *DelayedQueue {
	item, exists := u.itemMap[req.URL.Hostname()]
	if !exists {
		queue := &DelayedQueue{queue: []*Request{}}
		item = &PQueueItem{value: queue, priority: int(queue.clock.Unix())}
		u.itemMap[req.URL.Hostname()] = item
		heap.Push(u.queue, item)
	} else if item.value.(*DelayedQueue).Len() == 0 {
		heap.Push(u.queue, item)
	}

	return item.value.(*DelayedQueue)
}

func (u *URLQueue) Len() int {
	u.Lock()
	defer u.Unlock()

	return u.queue.Len()
}
