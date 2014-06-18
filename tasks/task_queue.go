// Package tasks provides ...
package tasks

import (
	"sync"
	"time"

	"container/heap"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
)

// TaskQueue represents a module that encapsulates a task queue
// A task queue defines all the methods needed for the task manager to
// schedule jobs, but the actual interface is actually pretty small
// and could service as interface for other types of queues too
type TaskQueue interface {
	exeggutor.Module
	// Enqueue puts an item on the queue
	Enqueue(item *protocol.ScheduledAppComponent) error
	// Dequeue pops the first item of the queue
	Dequeue() (*protocol.ScheduledAppComponent, error)
	// DequeueFirst pops the first item of the queue that matches the predicated
	DequeueFirst(func(*protocol.ScheduledAppComponent) bool) (*protocol.ScheduledAppComponent, error)
	// Len returns the size of the queue
	Len() int
}

// taskQueue represents the default implementation of a task queue
// The implementation of the queue uses a heap to manage a priority queue
// In this implementation the priorty queue favors highest cpu needs over
// highest memory needs over least recently added to the queue.
type taskQueue struct {
	pQueue *prioQueue
	lock   sync.Locker
}

// NewTaskQueue creates a new instance of the default task queue with
// an empty priority queue as storage
func NewTaskQueue() TaskQueue {
	return NewTaskQueueWithprioQueue(&prioQueue{})
}

// NewTaskQueueWithprioQueue creates a new instance of a default task queue
// with the provided priority queue as storage (mainly used for testing)
func NewTaskQueueWithprioQueue(q *prioQueue) TaskQueue {
	tq := &taskQueue{pQueue: q, lock: &sync.Mutex{}}
	heap.Init(tq.pQueue)
	return tq
}

// Start starts this component, acquiring a database etc when backed with
// a persistent priority queue
func (tq *taskQueue) Start() error {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	return nil
}

// Stop stops this component, releasing any resources it might be holding on to
func (tq *taskQueue) Stop() error {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	return nil
}

// Len returns the size of the queue
func (tq *taskQueue) Len() int {
	return tq.pQueue.Len()
}

// Enqueue enqueues an item if it hasn't been queued already
func (tq *taskQueue) Enqueue(item *protocol.ScheduledAppComponent) error {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	heap.Push(tq.pQueue, item)
	return nil
}

// Dequeue dequeues an item from the queue
func (tq *taskQueue) Dequeue() (*protocol.ScheduledAppComponent, error) {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	item := heap.Pop(tq.pQueue)
	return item.(*protocol.ScheduledAppComponent), nil
}

// DequeueFirst dequeues the first item from the queue that matches the predicated
func (tq *taskQueue) DequeueFirst(shouldDequeue func(*protocol.ScheduledAppComponent) bool) (*protocol.ScheduledAppComponent, error) {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	queue := *tq.pQueue
	var found *protocol.ScheduledAppComponent
	for _, item := range queue {
		if shouldDequeue(item) {
			found = item
			heap.Remove(tq.pQueue, int(item.GetPosition()))
			break
		}
	}
	return found, nil
}

// prioQueue a type to represent the default priority queue
type prioQueue []*protocol.ScheduledAppComponent

// Len returns the size of this priority queue
func (pq prioQueue) Len() int {
	return len(pq)
}

func (pq prioQueue) byCpu(left, right *protocol.ApplicationComponent) bool {
	return left.GetCpus() > right.GetCpus()
}

func (pq prioQueue) byMemorySecondary(left, right *protocol.ApplicationComponent) bool {
	return left.GetCpus() == right.GetCpus() && left.GetMem() > right.GetMem()
}

func (pq prioQueue) leastRecent(left, right *protocol.ScheduledAppComponent) bool {
	lcomp, rcomp := left.Component, right.Component
	return lcomp.GetCpus() == rcomp.GetCpus() && lcomp.GetMem() == rcomp.GetMem() && left.GetSince() < right.GetSince()
}

// Less returns true when the item at index i
// is higher on the list than the item at index j
func (pq prioQueue) Less(i, j int) bool {
	left, right := pq[i], pq[j]
	return pq.byCpu(left.Component, right.Component) ||
		pq.byMemorySecondary(left.Component, right.Component) ||
		pq.leastRecent(left, right)
}

// Swap swaps 2 items in the queue from position.
func (pq prioQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Position = proto.Int(i)
	pq[j].Position = proto.Int(j)
}

// Push pushes a new item onto this queue
func (pq *prioQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*protocol.ScheduledAppComponent)
	item.Position = proto.Int(n)
	item.Since = proto.Int64(time.Now().UTC().UnixNano())
	*pq = append(*pq, item)
}

// Pop pops a new item of the queue
func (pq *prioQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Position = proto.Int(-1)
	item.Since = proto.Int64(-1)
	*pq = old[0 : n-1]
	return item
}
