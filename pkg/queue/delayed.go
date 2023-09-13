package queue

import (
	"container/heap"
	"time"
)

type DelayedQueue struct {
    pending chan any
    delay time.Duration
}

func NewDelayedQueue(size int, delay time.Duration) *DelayedQueue {
    return &DelayedQueue{
        pending: make(chan any, size),
        delay: delay,
    }
}

func (dq *DelayedQueue) Enqueue(item any) {
    dq.pending<-item
}

func (dq *DelayedQueue) Poll(out chan any) {
    for {
        select {
        case item := <-dq.pending:
            time.Sleep(dq.delay)
            out<-item
        case <-time.After(dq.delay):
            return
        }
    }
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
