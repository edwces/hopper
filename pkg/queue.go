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

type TimeItem struct {
	value    any
	priority time.Time
	index    int
}

type TimeQueue []*TimeItem

// Len returns size of priority queue.
func (tq TimeQueue) Len() int {
	return len(tq)
}

// Less returns true if Item with index j has lower priority than
// Item with index i.
func (tq TimeQueue) Less(i, j int) bool {
	return tq[i].priority.Before(tq[j].priority)
}

// Swap swaps heap items with indexes i, j.
func (tq TimeQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].index = i
	tq[j].index = j
}

// Push appends an Item to the heap.
func (tq *TimeQueue) Push(x any) {
	n := len(*tq)
	item := x.(*TimeItem)
	item.index = n
	*tq = append(*tq, item)
}

// Update changes item priority and value.
func (tq *TimeQueue) Update(item *TimeItem, value any, priority time.Time) {
	item.value = value
	item.priority = priority
	heap.Fix(tq, item.index)
}

// Pop removes and returns item with a highest priority.
func (tq *TimeQueue) Pop() any {
	n := len(*tq)
	full := *tq
	popped := full[n-1]
	full[n-1] = nil
	*tq = full[:n-1]
	return popped
}

func (tq *TimeQueue) Peek() any {
	full := *tq
	return full[0]
}

type MemoryFrontier struct {
	hostQueue *TimeQueue
	hostMap   map[string]*TimeItem
	size      int
}

type HostQueue struct {
	lastReq time.Time

	uriQueue *PQueue
}

// Init heapifies all items in queue.
func (mf *MemoryFrontier) Init(rawUrls ...string) {
	mf.hostQueue = &TimeQueue{}
	mf.hostMap = map[string]*TimeItem{}
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
		hostItem = &TimeItem{value: hostQueue, priority: time.Now()}
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
	hostItem := mf.hostQueue.Peek().(*TimeItem)
	hostQueue := hostItem.value.(*HostQueue)

	time.Sleep(time.Until(hostQueue.lastReq.Add(DefaultDelay)))

	// update time of request
	hostQueue.lastReq = time.Now()
	mf.hostQueue.Update(hostItem, hostItem.value, time.Now().Add(DefaultDelay))

	uriItem := heap.Pop(hostQueue.uriQueue)
	mf.size--

	return uriItem.(*Item).value.(string)
}

func (mf *MemoryFrontier) Len() int {
	return mf.size
}
