package crawler

import (
	"container/heap"
	"net/url"
	"sync"
	"time"
)

// Frontier is used for storage and retrievment of crawled urls
type Frontier struct {
	sync.Mutex

	Delay time.Duration

	// needs to be PQueue so we can get the hostQueue with earliest request in pop
	queue *PQueue
	// used for fast access of hostQueues when pushing new value
	hostMap map[string]*HostQueue
}

type HostQueue struct {
	name    string
	lastReq time.Time

	queue *PQueue
}

func (f *Frontier) Init(seeds ...string) {
	f.queue = &PQueue{}
	heap.Init(f.queue)
	if f.Delay == 0 {
		f.Delay = time.Second
	}

	for _, seed := range seeds {
		f.Push(seed)
	}
}

func (f *Frontier) Push(rawUrl string) error {
	uri, err := url.Parse(rawUrl)

	if err != nil {
		return err
	}

	f.Lock()
	defer f.Unlock()

	if f.hostMap[uri.Host] == nil {
		urlQueue := &PQueue{}
		heap.Init(urlQueue)
		hostQueue := HostQueue{name: uri.Host, queue: urlQueue, lastReq: time.Now().Add(-f.Delay)}
		heap.Push(f.queue, &hostQueue)
		f.hostMap[uri.Host] = &hostQueue
	}

	hostQueue := f.hostMap[uri.Host]
	heap.Push(hostQueue.queue, &Item{value: rawUrl, priority: 1})
	return nil
}

func (f *Frontier) Pop() *HostQueue {
	// Pop maybe should be blocking until delay
	// get host queue with earliest last request
	f.Lock()
	defer f.Unlock()

	hostQueue := f.queue.Peek().(*HostQueue)
	time.Sleep(time.Until(hostQueue.lastReq))

	if f.queue.Len() == 0 {
		heap.Pop(f.queue)
		f.hostMap[hostQueue.name] = nil
		delete(f.hostMap, hostQueue.name)
	}

	return hostQueue
}
