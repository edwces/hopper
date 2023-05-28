package hopper

import (
	"container/heap"
	"net/url"
	"time"
)

type PQHeapItem struct {
	value    any
	priority any
	index    int
}

type PQueueHeap struct {
	heap        []*PQHeapItem
	compareFunc func(x any, y any) bool
}

// Len returns size of priority queue.
func (pqh PQueueHeap) Len() int {
	return len(pqh.heap)
}

// Less returns true if Item with index j has lower priority than
// Item with index i.
func (pqh PQueueHeap) Less(i, j int) bool {
	return pqh.compareFunc(pqh.heap[i].priority, pqh.heap[j].priority)
}

// Swap swaps heap items with indexes i, j.
func (pqh PQueueHeap) Swap(i, j int) {
	pqh.heap[i], pqh.heap[j] = pqh.heap[j], pqh.heap[i]
	pqh.heap[i].index = i
	pqh.heap[j].index = j
}

// Push appends an Item to the heap.
func (pqh *PQueueHeap) Push(x any) {
	n := len(pqh.heap)
	item := x.(*PQHeapItem)
	item.index = n
	pqh.heap = append(pqh.heap, item)
}

// Update changes item priority and value.
func (pqh *PQueueHeap) Update(item *PQHeapItem, value any, priority any) {
	item.value = value
	item.priority = priority
	heap.Fix(pqh, item.index)
}

// Pop removes and returns item with a highest priority.
func (pqh *PQueueHeap) Pop() any {
	n := len(pqh.heap)
	full := pqh.heap
	popped := full[n-1]
	full[n-1] = nil
	pqh.heap = full[:n-1]
	return popped
}

type PQueue struct {
	pqheap *PQueueHeap
}

func NewPQueue(compareFunc func(x any, y any) bool) *PQueue {
	pq := PQueue{}
	pq.pqheap = &PQueueHeap{compareFunc: compareFunc}
	heap.Init(pq.pqheap)
	return &pq
}

func (pq *PQueue) Push(item *PQHeapItem) {
	heap.Push(pq.pqheap, item)
}

func (pq *PQueue) Pop() *PQHeapItem {
	return heap.Pop(pq.pqheap).(*PQHeapItem)
}

func (pq *PQueue) Peek() *PQHeapItem {
	return pq.pqheap.heap[0]
}

func (pq *PQueue) Update(item *PQHeapItem, value any, priority any) {
	pq.pqheap.Update(item, value, priority)
}

type InMemoryURLQueue struct {
	Delay time.Duration

	hostQueue *PQueue
	hostMap   map[string]*PQHeapItem
	size      int
}

type URLHost struct {
	Delay time.Duration

	queue []string
}

// Init initializes queue with given urls.
func (muq *InMemoryURLQueue) Init(uris ...*url.URL) {
	muq.hostQueue = NewPQueue(func(x, y any) bool { return x.(time.Time).Before(y.(time.Time)) })
	muq.hostMap = map[string]*PQHeapItem{}

	for _, uri := range uris {
		muq.Push(uri)
	}
}

// Push adds uri to the queue.
func (muq *InMemoryURLQueue) Push(uri *url.URL) error {
	// check if URLHost for given url exists
	urlHost, exists := muq.hostMap[uri.Host]
	if !exists {
		urlHostValue := &URLHost{queue: []string{}, Delay: muq.Delay}
		urlHost = &PQHeapItem{value: urlHostValue, priority: time.Now()}
		muq.hostQueue.Push(urlHost)
		muq.hostMap[uri.Host] = urlHost
	}

	urlHost.value.(*URLHost).queue = append(urlHost.value.(*URLHost).queue, uri.String())
	muq.size++

	return nil
}

// Pop returns and removes item which is scheduled the latest.
// It also waits the specified delay for given url host.
func (muq *InMemoryURLQueue) Pop() string {
	urlHost := muq.hostQueue.Peek()
	urlHostValue := urlHost.value.(*URLHost)

	time.Sleep(time.Until(urlHost.priority.(time.Time)))
	muq.hostQueue.Update(urlHost, urlHost.value, time.Now().Add(urlHostValue.Delay))

	n := len(urlHostValue.queue)
	uri := urlHostValue.queue[n-1]
	urlHostValue.queue = urlHostValue.queue[:n-1]
	muq.size--

	return uri
}

// Len returns length of overall uri items in all hosts.
func (muq *InMemoryURLQueue) Len() int {
	return muq.size
}

// Update changes URLHost Delay.
func (muq *InMemoryURLQueue) Update(host string, delay time.Duration) {
	hostItem, exists := muq.hostMap[host]
	if exists && delay != 0 {
		hostItem.value.(*URLHost).Delay = delay
	}
}
