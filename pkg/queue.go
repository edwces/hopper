package crawler

import (
	"container/heap"
	"sync"
)

type Item struct {
	value    any
	priority int
}

type PQueue []*Item

// TODO: Improve Permormance
type SafePQueue struct {
	sync.RWMutex

	queue heap.Interface
}

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
}

// Push appends an Item to the heap.
func (pq *PQueue) Push(x any) {
	item := x.(*Item)
	*pq = append(*pq, item)
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

// Init heapifies all items in queue.
func (spq *SafePQueue) Init(items ...*Item) {
	pq := PQueue{}
	copy(pq, items)
	spq.queue = &pq
	heap.Init(spq.queue)
}

// Push safely adds item to queue.
func (spq *SafePQueue) Push(x Item) {
	spq.Lock()
	defer spq.Unlock()
	heap.Push(spq.queue, x)
}

// Pop returns and removes item with highest priority
func (spq *SafePQueue) Pop() *Item {
	spq.Lock()
	defer spq.Unlock()
	return heap.Pop(spq.queue).(*Item)
}

// Len returns size of priority queue.
func (spq *SafePQueue) Len() int {
	spq.RLock()
	defer spq.RUnlock()
	return spq.queue.Len()
}
