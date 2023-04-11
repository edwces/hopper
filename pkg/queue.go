package crawler

type Node[T comparable] struct {
	value T
	next  *Node[T]
}

type Queue[T comparable] struct {
	head *Node[T]
	tail *Node[T]
	size int
}

func NewQueue[T comparable](values ...T) *Queue[T] {
	qp := &Queue[T]{}
	for _, value := range values {
		qp.Enqueue(value)
	}
	return qp
}

func (qp *Queue[T]) Dequeue() T {
	if qp.head == nil {
		return *new(T)
	}
	popped := qp.head.value
	qp.head = qp.head.next
	qp.size--
	return popped
}

func (qp *Queue[T]) Enqueue(value T) {
	n := &Node[T]{value: value, next: nil}
	if qp.head == nil {
		qp.head = n
	} else if qp.tail == nil {
		qp.tail = n
	} else {
		qp.tail.next = n
	}
	qp.size++
}
