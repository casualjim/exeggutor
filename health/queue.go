package health

import (
	"container/heap"
	"sync"
	"time"

	"github.com/reverb/exeggutor/health/check"
)

// activeHealthCheck represents a scheduled health check
// this has a position in the health check queue based on its expiration
type activeHealthCheck struct {
	check.HealthCheck
	ExpiresAt time.Time
	index     int
}

type healthCheckPQueue []*activeHealthCheck

func (h healthCheckPQueue) Len() int {
	return len(h)
}

func (h healthCheckPQueue) Less(i, j int) bool {
	if h.Len() == 0 {
		return true
	}
	left, right := h[i], h[j]
	return left.ExpiresAt.Before(right.ExpiresAt)
}

func (h healthCheckPQueue) Swap(i, j int) {
	if len(h) == 0 {
		return
	}
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *healthCheckPQueue) Push(x interface{}) {
	n := len(*h)
	item := x.(*activeHealthCheck)
	item.index = n
	*h = append(*h, item)
}

func (h *healthCheckPQueue) Pop() interface{} {
	old := *h
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	item.index = -1
	*h = old[0 : n-1]
	return item
}

type healthCheckQueue struct {
	queue healthCheckPQueue
	lock  *sync.RWMutex
}

func newHealthCheckQueue() *healthCheckQueue {
	q := healthCheckPQueue{}
	heap.Init(&q)
	return &healthCheckQueue{queue: q, lock: &sync.RWMutex{}}
}

func (h *healthCheckQueue) Len() int {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.queue.Len()
}

func (h *healthCheckQueue) Push(hc *activeHealthCheck) {
	h.lock.Lock()
	defer h.lock.Unlock()
	heap.Push(&h.queue, hc)
}

func (h *healthCheckQueue) Pop() (*activeHealthCheck, time.Time, bool) {
	h.lock.Lock()
	defer h.lock.Unlock()
	item := heap.Pop(&h.queue)
	if item == nil {
		return nil, time.Now().Add(1 * time.Second), false
	}
	ac := item.(*activeHealthCheck)
	if ac.ExpiresAt.After(time.Now()) {
		heap.Push(&h.queue, ac)
		return nil, ac.ExpiresAt, false
	}
	return ac, time.Now(), true
}

func (h *healthCheckQueue) Remove(id string) {
	h.lock.Lock()
	defer h.lock.Unlock()
	for i, ac := range h.queue {
		if ac.GetID() == id {
			heap.Remove(&h.queue, i)
			break
		}
	}
}

func (h *healthCheckQueue) Contains(id string) bool {
	for _, chk := range h.queue {
		if chk.GetID() == id {
			return true
		}
	}
	return false
}
