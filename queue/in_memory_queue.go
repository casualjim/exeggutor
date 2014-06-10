package queue

import "sync"

type inMemoryQueuNode struct {
	data interface{}
	next *inMemoryQueuNode
}

// InMemoryQueue An in-memory FIFO queue datastructure.
// has O(1) for inserting things into the queue, it does that by using a linked list.
type InMemoryQueue struct {
	head  *inMemoryQueuNode
	tail  *inMemoryQueuNode
	count int
	lock  *sync.Mutex
}

// NewInMemoryQueue creates a new instance of an InMemoryQueue
func NewInMemoryQueue() *InMemoryQueue {
	q := &InMemoryQueue{}
	q.lock = &sync.Mutex{}
	return q
}

// Len returns the length of the queue
func (q *InMemoryQueue) Len() (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.count, nil
}

// Push enqueues the provided item
func (q *InMemoryQueue) Push(item interface{}) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	n := &inMemoryQueuNode{data: item}

	if q.tail == nil {
		q.tail = n
		q.head = n
	} else {
		q.tail.next = n
		q.tail = n
	}
	q.count++
	return nil
}

// Poll dequeues an item from the queue if there is one
// returns (nil, nil) if there is no item on the queue!
func (q *InMemoryQueue) Poll() (interface{}, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.head == nil {
		return nil, nil
	}

	n := q.head
	q.head = n.next

	if q.head == nil {
		q.tail = nil
	}
	q.count--

	return n.data, nil
}

// Peek look at the next item to dequeue, but don't dequeue
func (q *InMemoryQueue) Peek() (interface{}, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	n := q.head
	if n == nil || n.data == nil {
		return nil, nil
	}

	return n.data, nil
}

// Start start this queue
func (q *InMemoryQueue) Start() error {
	return nil
}

// Stop stops this queue, resetting the state of this queue
func (q *InMemoryQueue) Stop() error {
	q.count = 0
	q.head = nil
	q.tail = nil
	return nil
}
