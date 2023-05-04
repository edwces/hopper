package hopper

import (
	"container/heap"
	"net/url"
	"time"
)

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
	full := *pq
	return full[0]
}

type MemoryFrontier struct {
	hostQueue *PQueue
	hostMap   map[string]*Item
	size      int
}

type HostQueue struct {
	lastReq time.Time

	uriQueue *PQueue
}

// Init heapifies all items in queue.
func (mf *MemoryFrontier) Init(rawUrls ...string) {
	mf.hostQueue = &PQueue{}
	mf.hostMap = map[string]*Item{}
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

	// check if hostQueue for given url exists
	hostItem, exists := mf.hostMap[uri.Host]
	if !exists {
		uriQueue := &PQueue{}
		heap.Init(uriQueue)
		hostQueue := &HostQueue{uriQueue: uriQueue, lastReq: time.Now().Add(-DefaultDelay)}
		hostItem = &Item{value: hostQueue, priority: 2}
		heap.Push(mf.hostQueue, hostItem)
		mf.hostMap[uri.Host] = hostItem
	}

	uriItem := &Item{value: rawUrl, priority: 1}
	heap.Push(hostItem.value.(*HostQueue).uriQueue, uriItem)
	mf.size++

	return nil
}

// Pop returns and removes item with highest priority.
// It also waits the specified delay for given url host.
func (mf *MemoryFrontier) Pop() string {
	hostItem := mf.hostQueue.Peek().(*Item)
	hostQueue := hostItem.value.(*HostQueue)

	time.Sleep(time.Until(hostQueue.lastReq.Add(DefaultDelay)))

	// update time of request
	hostQueue.lastReq = time.Now()
	mf.hostQueue.Update(hostItem, hostItem.value, 1)

	uriItem := heap.Pop(hostQueue.uriQueue)
	mf.size--

	return uriItem.(*Item).value.(string)
}

func (mf *MemoryFrontier) Len() int {
	return mf.size
}
