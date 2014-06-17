// Package tasks provides ...
package tasks

import (
	"time"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor/protocol"
)

type TaskQueue []*protocol.ScheduledAppComponent

func (tq TaskQueue) Len() int {
	return len(tq)
}

func (tq TaskQueue) byCpu(left, right *protocol.ApplicationComponent) bool {
	return left.GetCpus() > right.GetCpus()
}

func (tq TaskQueue) byMemorySecondary(left, right *protocol.ApplicationComponent) bool {
	return left.GetCpus() == right.GetCpus() && left.GetMem() > right.GetMem()
}

func (tq TaskQueue) leastRecent(left, right *protocol.ScheduledAppComponent) bool {
	lcomp, rcomp := left.Component, right.Component
	return lcomp.GetCpus() == rcomp.GetCpus() && lcomp.GetMem() == rcomp.GetMem() && left.GetSince() < right.GetSince()
}

func (tq TaskQueue) Less(i, j int) bool {
	left, right := tq[i], tq[j]
	return tq.byCpu(left.Component, right.Component) ||
		tq.byMemorySecondary(left.Component, right.Component) ||
		tq.leastRecent(left, right)
}

func (tq TaskQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].Position = proto.Int(i)
	tq[j].Position = proto.Int(j)
}

func (tq *TaskQueue) Push(x interface{}) {
	n := len(*tq)
	item := x.(*protocol.ScheduledAppComponent)
	item.Position = proto.Int(n)
	item.Since = proto.Int64(time.Now().UTC().UnixNano())
	*tq = append(*tq, item)
}

func (tq *TaskQueue) Pop() interface{} {
	old := *tq
	n := len(old)
	item := old[n-1]
	item.Position = proto.Int(-1)
	item.Since = proto.Int64(-1)
	*tq = old[0 : n-1]
	return item
}
