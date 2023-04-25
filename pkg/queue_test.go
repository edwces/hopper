package hopper

import (
	"container/heap"
	"math/rand"
	"testing"
)

func generateRandString(n int) string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321")
	str := make([]rune, n)
	for i := range str {
		str[i] = chars[rand.Intn(len(chars))]
	}
	return string(str)
}

func TestPQueueLen(t *testing.T) {
	pq := &PQueue{}

	heap.Init(pq)
	item := &Item{value: "hello", priority: 3}
	item2 := &Item{value: "new", priority: 1}
	item3 := &Item{value: "world", priority: 2}
	item4 := &Item{value: "another", priority: 2}
	heap.Push(pq, item)
	heap.Push(pq, item2)
	heap.Push(pq, item3)
	heap.Push(pq, item4)

	ln := pq.Len()
	if ln != 4 {
		t.Errorf("pq.Len() = %d, want %d", ln, 4)
	}

	heap.Pop(pq)
	heap.Pop(pq)
	ln = pq.Len()
	if ln != 2 {
		t.Errorf("pq.Len() = %d, want %d", ln, 2)
	}

	heap.Remove(pq, 1)
	ln = pq.Len()
	if ln != 1 {
		t.Errorf("pq.Len() = %d, want %d", ln, 1)
	}
}

func TestPQueuePop(t *testing.T) {
	pq := &PQueue{}

	heap.Init(pq)
	item := &Item{value: "good", priority: 5}
	item3 := &Item{value: "world", priority: 2}
	item4 := &Item{value: "another", priority: 2}
	item5 := &Item{value: "something", priority: 4}
	heap.Push(pq, item)
	heap.Push(pq, item3)
	heap.Push(pq, item4)
	heap.Push(pq, item5)

	popped := heap.Pop(pq).(*Item)
	if popped != item {
		t.Errorf("heap.Pop() = %+v, want %+v", popped, item)
	}

	popped = heap.Pop(pq).(*Item)
	if popped != item5 {
		t.Errorf("heap.Pop() = %+v, want %+v", popped, item5)
	}

	item6 := &Item{value: "After Added", priority: 10}
	heap.Push(pq, item6)
	popped = heap.Pop(pq).(*Item)
	if popped != item6 {
		t.Errorf("heap.Pop() = %+v, want %+v", popped, item6)
	}

	popped1 := heap.Pop(pq).(*Item)
	popped2 := heap.Pop(pq).(*Item)
	if !(popped1 == item3 && popped2 == item4) && !(popped1 == item4 && popped2 == item3) {
		t.Errorf("heap.Pop(), heap.Pop() = (%+v, %+v), want (%+v, %+v) or (%+v, %+v)",
			popped1, popped2, item3, item4, item4, item3)
	}
}

func BenchmarkQueuePush(b *testing.B) {
	pq := &PQueue{}
	heap.Init(pq)

	for i := 0; i < b.N; i++ {
		randItem := &Item{value: generateRandString(rand.Intn(25)), priority: rand.Intn(1000)}
		heap.Push(pq, randItem)
	}
}

func BenchmarkQueuePop(b *testing.B) {
	pq := &PQueue{}
	heap.Init(pq)

	for i := 0; i < b.N; i++ {
		randItem := &Item{value: generateRandString(rand.Intn(25)), priority: rand.Intn(1000)}
		heap.Push(pq, randItem)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		heap.Pop(pq)
	}
}
