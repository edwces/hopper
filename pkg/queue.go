package hopper

import (
	"container/heap"
	"net/url"
	"sync"
	"time"
)

type Frontier interface {
	Init(rawUrls ...string)
	Push(uri string) error
	Pop() *Item
}

type Item struct {
	value    any
	priority int
	index    int
}

type PQueue []*Item

// Len returns size of priority queue.
func (pq PQueue) Len() int {
	return len(pq)
}

// Less returns true if Item with index j has lower priority than
// Item with index i.
func (pq PQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

// Swap swaps heap items with indexes i, j.
func (pq PQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push appends an Item to the heap.
func (pq *PQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

// Update changes item priority and value.
func (pq *PQueue) Update(item *Item, value any, priority int) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}

// Pop removes and returns item with a highest priority.
func (pq *PQueue) Pop() any {
	n := len(*pq)
	full := *pq
	popped := full[n-1]
	full[n-1] = nil
	*pq = full[:n-1]
	return popped
}

func (pq *PQueue) Peek() any {
	n := len(*pq)
	full := *pq
	return full[n-1]
}

type MemoryFrontier struct {
	sync.RWMutex

	hostQueue *PQueue
	hostMap   map[string]*Item
}

type HostQueue struct {
	lastReq time.Time

	uriQueue *PQueue
}

// Init heapifies all items in queue.
func (mf *MemoryFrontier) Init(rawUrls ...string) {
	mf.hostQueue = &PQueue{}
	heap.Init(mf.hostQueue)
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

	mf.Lock()
	defer mf.Unlock()
	// check if hostQueue for given url exists
	hostItem, exists := mf.hostMap[uri.Host]
	if !exists {
		uriQueue := &PQueue{}
		heap.Init(uriQueue)
		hostQueue := &HostQueue{uriQueue: uriQueue, lastReq: time.Now().Add(-DefaultDelay)}
		hostItem = &Item{value: hostQueue, priority: 2}
		heap.Push(mf.hostQueue, hostItem)
	}

	uriItem := &Item{value: rawUrl, priority: 1}
	heap.Push(hostItem.value.(*HostQueue).uriQueue, uriItem)

	return nil
}

// Pop returns and removes item with highest priority
func (mf *MemoryFrontier) Pop() *Item {
	mf.Lock()
	hostItem := mf.hostQueue.Peek().(*Item)
	hostQueue := hostItem.value.(*HostQueue)
	mf.Unlock()

	time.Sleep(time.Until(hostQueue.lastReq))

	// update time of request
	mf.Lock()
	defer mf.Unlock()
	hostQueue.lastReq = time.Now()
	mf.hostQueue.Update(hostItem, hostItem.value, 1)

	uriItem := heap.Pop(hostQueue.uriQueue)
	return uriItem.(*Item)
}
