// Package tasks provides ...
package tasks

import (
	"time"

	"container/heap"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
)

type TaskQueue interface {
	exeggutor.Module
	Enqueue(item *protocol.ScheduledAppComponent) error
	Dequeue() (*protocol.ScheduledAppComponent, error)
	DequeueFirst(func(*protocol.ScheduledAppComponent) bool) (*protocol.ScheduledAppComponent, error)
	Len() int
}

type DefaultTaskQueue struct {
	pQueue *PrioQueue
}

func NewTaskQueue() TaskQueue {
	return NewTaskQueueWithPrioQueue(&PrioQueue{})
}

func NewTaskQueueWithPrioQueue(q *PrioQueue) TaskQueue {
	tq := &DefaultTaskQueue{pQueue: q}
	heap.Init(tq.pQueue)
	return tq
}

func (tq *DefaultTaskQueue) Start() error {
	return nil
}

func (tq *DefaultTaskQueue) Stop() error {
	return nil
}

func (tq *DefaultTaskQueue) Len() int {
	return tq.pQueue.Len()
}

func (tq *DefaultTaskQueue) Enqueue(item *protocol.ScheduledAppComponent) error {
	heap.Push(tq.pQueue, item)
	return nil
}

func (tq *DefaultTaskQueue) Dequeue() (*protocol.ScheduledAppComponent, error) {
	item := heap.Pop(tq.pQueue)
	return item.(*protocol.ScheduledAppComponent), nil
}

func (tq *DefaultTaskQueue) DequeueFirst(shouldDequeue func(*protocol.ScheduledAppComponent) bool) (*protocol.ScheduledAppComponent, error) {
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

type PrioQueue []*protocol.ScheduledAppComponent

func (pq PrioQueue) Len() int {
	return len(pq)
}

func (pq PrioQueue) byCpu(left, right *protocol.ApplicationComponent) bool {
	return left.GetCpus() > right.GetCpus()
}

func (pq PrioQueue) byMemorySecondary(left, right *protocol.ApplicationComponent) bool {
	return left.GetCpus() == right.GetCpus() && left.GetMem() > right.GetMem()
}

func (pq PrioQueue) leastRecent(left, right *protocol.ScheduledAppComponent) bool {
	lcomp, rcomp := left.Component, right.Component
	return lcomp.GetCpus() == rcomp.GetCpus() && lcomp.GetMem() == rcomp.GetMem() && left.GetSince() < right.GetSince()
}

func (pq PrioQueue) Less(i, j int) bool {
	left, right := pq[i], pq[j]
	return pq.byCpu(left.Component, right.Component) ||
		pq.byMemorySecondary(left.Component, right.Component) ||
		pq.leastRecent(left, right)
}

func (pq PrioQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Position = proto.Int(i)
	pq[j].Position = proto.Int(j)
}

func (pq *PrioQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*protocol.ScheduledAppComponent)
	item.Position = proto.Int(n)
	item.Since = proto.Int64(time.Now().UTC().UnixNano())
	*pq = append(*pq, item)
}

func (pq *PrioQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Position = proto.Int(-1)
	item.Since = proto.Int64(-1)
	*pq = old[0 : n-1]
	return item
}
