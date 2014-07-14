// Package queue provides ...
package queue

import (
	"sync"
	"time"

	"container/heap"

	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
)

var log = logging.MustGetLogger("exeggutor.tasks.queue")

// TaskQueue represents a module that encapsulates a task queue
// A task queue defines all the methods needed for the task manager to
// schedule jobs, but the actual interface is actually pretty small
// and could service as interface for other types of queues too
type TaskQueue interface {
	exeggutor.Module
	// Enqueue puts an item on the queue
	Enqueue(item *protocol.ScheduledApp) error
	// Dequeue pops the first item of the queue
	Dequeue() (*protocol.ScheduledApp, error)
	// DequeueFirst pops the first item of the queue that matches the predicated
	DequeueFirst(func(*protocol.ScheduledApp) bool) (*protocol.ScheduledApp, error)
	// CountAppsForId returns how many apps are currently scheduled for deployment
	// with that specified app id
	CountAppsForID(appID string) int32
	// CountsForApps
	CountsForApps() map[string]int32
	// Len returns the size of the queue
	Len() int
}

// taskQueue represents the default implementation of a task queue
// The implementation of the queue uses a heap to manage a priority queue
// In this implementation the priorty queue favors highest cpu needs over
// highest memory needs over least recently added to the queue.
type taskQueue struct {
	pQueue *PrioQueue
	lock   sync.Locker
}

// New creates a new instance of the default task queue with
// an empty priority queue as storage
func New() TaskQueue {
	return NewTaskQueueWithPrioQueue(&PrioQueue{})
}

// NewTaskQueueWithPrioQueue creates a new instance of a default task queue
// with the provided priority queue as storage (mainly used for testing)
func NewTaskQueueWithPrioQueue(q *PrioQueue) TaskQueue {
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

// CountAppsForId returns how many apps are currently scheduled for deployment
// with that specified app id
func (tq *taskQueue) CountAppsForID(appID string) int32 {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	queue := *tq.pQueue

	var count int32
	for _, item := range queue {
		if item.GetAppId() == appID {
			count++
		}
	}
	return count
}

func (tq *taskQueue) CountsForApps() map[string]int32 {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	queue := *tq.pQueue
	var counts map[string]int32
	for _, item := range queue {
		appID := item.GetAppId()
		cnt, ok := counts[appID]
		if !ok {
			cnt = 0
		}
		counts[appID] = cnt + 1
	}
	return counts
}

// Enqueue enqueues an item if it hasn't been queued already
func (tq *taskQueue) Enqueue(item *protocol.ScheduledApp) error {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	heap.Push(tq.pQueue, item)
	return nil
}

// Dequeue dequeues an item from the queue
func (tq *taskQueue) Dequeue() (*protocol.ScheduledApp, error) {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	item := heap.Pop(tq.pQueue)
	return item.(*protocol.ScheduledApp), nil
}

// DequeueFirst dequeues the first item from the queue that matches the predicated
func (tq *taskQueue) DequeueFirst(shouldDequeue func(*protocol.ScheduledApp) bool) (*protocol.ScheduledApp, error) {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	queue := *tq.pQueue
	var found *protocol.ScheduledApp
	for _, item := range queue {
		if shouldDequeue(item) {
			found = item
			heap.Remove(tq.pQueue, int(item.GetPosition()))
			break
		}
	}
	return found, nil
}

// PrioQueue a type to represent the default priority queue
type PrioQueue []*protocol.ScheduledApp

// Len returns the size of this priority queue
func (pq PrioQueue) Len() int {
	return len(pq)
}

func (pq PrioQueue) byCPU(left, right *protocol.Application) bool {
	return left.GetCpus() > right.GetCpus()
}

func (pq PrioQueue) byMemorySecondary(left, right *protocol.Application) bool {
	return left.GetCpus() == right.GetCpus() && left.GetMem() > right.GetMem()
}

func (pq PrioQueue) leastRecent(left, right *protocol.ScheduledApp) bool {
	lcomp, rcomp := left.App, right.App
	return lcomp.GetCpus() == rcomp.GetCpus() && lcomp.GetMem() == rcomp.GetMem() && left.GetSince() < right.GetSince()
}

// Less returns true when the item at index i
// is higher on the list than the item at index j
func (pq PrioQueue) Less(i, j int) bool {
	left, right := pq[i], pq[j]
	return pq.byCPU(left.App, right.App) ||
		pq.byMemorySecondary(left.App, right.App) ||
		pq.leastRecent(left, right)
}

// Swap swaps 2 items in the queue from position.
func (pq PrioQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Position = proto.Int(i)
	pq[j].Position = proto.Int(j)
}

// Push pushes a new item onto this queue
func (pq *PrioQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*protocol.ScheduledApp)
	item.Position = proto.Int(n)
	item.Since = proto.Int64(time.Now().UTC().UnixNano())
	*pq = append(*pq, item)
}

// Pop pops a new item of the queue
func (pq *PrioQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Position = proto.Int(-1)
	item.Since = proto.Int64(-1)
	*pq = old[0 : n-1]
	return item
}
