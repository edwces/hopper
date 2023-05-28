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

type MemoryFrontier struct {
	Delay time.Duration

	hostQueue *PQueue
	hostMap   map[string]*PQHeapItem
	size      int
}

type HostQueue struct {
	Delay   time.Duration
	LastReq time.Time

	uriQueue *PQueue
}

// Init heapifies all items in queue.
func (mf *MemoryFrontier) Init(rawUrls ...string) {
	if mf.Delay == 0 {
		errorLogger.Fatal("frontier default delay has not been specified")
	}

	mf.hostQueue = NewPQueue(func(x, y any) bool { return x.(time.Time).Before(y.(time.Time)) })
	mf.hostMap = map[string]*PQHeapItem{}
	for _, rawUrl := range rawUrls {
		mf.Push(rawUrl)
	}
}

// Push safely adds item to queue.
func (mf *MemoryFrontier) Push(rawUrl string) error {

	uri, err := url.Parse(rawUrl)
	if err != nil {
		return err
	}

	// check if hostQueue for given url exists
	hostItem, exists := mf.hostMap[uri.Host]
	if !exists {
		uriQueue := NewPQueue(func(x, y any) bool { return x.(int) < y.(int) })
		hostQueue := &HostQueue{uriQueue: uriQueue, LastReq: time.Now().Add(-mf.Delay), Delay: mf.Delay}
		hostItem = &PQHeapItem{value: hostQueue, priority: time.Now()}
		mf.hostQueue.Push(hostItem)
		mf.hostMap[uri.Host] = hostItem
	}

	uriItem := &PQHeapItem{value: rawUrl, priority: 1}
	hostItem.value.(*HostQueue).uriQueue.Push(uriItem)
	mf.size++

	return nil
}

// Pop returns and removes item with highest priority.
// It also waits the specified delay for given url host.
func (mf *MemoryFrontier) Pop() string {
	hostItem := mf.hostQueue.Peek()
	hostQueue := hostItem.value.(*HostQueue)

	time.Sleep(time.Until(hostQueue.LastReq.Add(hostQueue.Delay)))

	// update time of request
	hostQueue.LastReq = time.Now()
	mf.hostQueue.Update(hostItem, hostItem.value, time.Now().Add(hostQueue.Delay))

	uriItem := hostQueue.uriQueue.Pop()
	mf.size--

	return uriItem.value.(string)
}

func (mf *MemoryFrontier) Len() int {
	return mf.size
}

func (mf *MemoryFrontier) Update(host string, delay time.Duration) {
	hostItem, exists := mf.hostMap[host]
	if exists {
		if delay == 0 {
			delay = hostItem.value.(*HostQueue).Delay
		}
		hostItem.value.(*HostQueue).LastReq = time.Now().Add(-delay)
		hostItem.value.(*HostQueue).Delay = delay
	}
}
