// Package tasks_test provides ...
package queue

import (
	"testing"

	"github.com/reverb/exeggutor/protocol"
	. "github.com/reverb/exeggutor/tasks"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTaskQueue(t *testing.T) {

	Convey("A DefaultTaskQueue", t, func() {

		q := &prioQueue{}
		tq := NewTaskQueueWithprioQueue(q)
		tq.Start()

		Reset(func() {
			tq.Stop()
		})

		Convey("when behaving as a queue", func() {

			Convey("should allow putting things onto the queue", func() {
				component := testComponent("app-tq-1", "comp-tq-1", 1.0, 64.0)
				scheduled := scheduledComponent(&component)
				err := tq.Enqueue(&scheduled)
				So(err, ShouldBeNil)
			})

			Convey("should allow popping things of the queue", func() {
				component := testComponent("app-tq-1", "comp-tq-1", 1.0, 64.0)
				scheduled := scheduledComponent(&component)
				cr := &scheduled
				tq.Enqueue(cr)

				component2 := testComponent("app-tq-2", "comp-tq-2", 1.0, 64.0)
				scheduled2 := scheduledComponent(&component2)
				cr2 := &scheduled2
				tq.Enqueue(cr2)

				item, err := tq.Dequeue()

				So(err, ShouldBeNil)
				So(item, ShouldResemble, cr)
				So(item, ShouldNotResemble, cr2)
				So(q.Len(), ShouldEqual, 1)
			})

			Convey("should allow picking the first thing that matches a predicate for dequeue", func() {
				component := testComponent("app-tq-1", "comp-tq-1", 1.0, 64.0)
				scheduled := scheduledComponent(&component)
				cr := &scheduled
				tq.Enqueue(cr)

				component2 := testComponent("app-tq-2", "comp-tq-2", 1.0, 64.0)
				scheduled2 := scheduledComponent(&component2)
				cr2 := &scheduled2
				tq.Enqueue(cr2)

				item, err := tq.DequeueFirst(func(item *protocol.ScheduledApp) bool {
					return item.GetAppName() == "app-tq-2"
				})

				So(err, ShouldBeNil)
				So(item, ShouldNotResemble, cr)
				So(item, ShouldResemble, cr2)
				So(q.Len(), ShouldEqual, 1)
			})
		})

		Convey("when being a priority queue", func() {

			Convey("should take the thing with the largest cpu needs first", func() {
				component := testComponent("app-tq-1", "comp-tq-1", 1.0, 64.0)
				scheduled := scheduledComponent(&component)
				cr := &scheduled
				tq.Enqueue(cr)

				component2 := testComponent("app-tq-2", "comp-tq-2", 1.0, 64.0)
				scheduled2 := scheduledComponent(&component2)
				cr2 := &scheduled2
				tq.Enqueue(cr2)

				component3 := testComponent("app-tq-3", "comp-tq-3", 1.5, 64.0)
				scheduled3 := scheduledComponent(&component3)
				cr3 := &scheduled3
				tq.Enqueue(cr3)

				item, err := tq.Dequeue()

				So(err, ShouldBeNil)
				So(item, ShouldResemble, cr3)
				So(q.Len(), ShouldEqual, 2)
			})

			Convey("should take the thing with the largest memory needs second", func() {
				component := testComponent("app-tq-1", "comp-tq-1", 1.0, 64.0)
				scheduled := scheduledComponent(&component)
				cr := &scheduled
				tq.Enqueue(cr)

				component2 := testComponent("app-tq-2", "comp-tq-2", 1.0, 128.0)
				scheduled2 := scheduledComponent(&component2)
				cr2 := &scheduled2
				tq.Enqueue(cr2)

				component3 := testComponent("app-tq-3", "comp-tq-3", 0.5, 64.0)
				scheduled3 := scheduledComponent(&component3)
				cr3 := &scheduled3
				tq.Enqueue(cr3)

				item, err := tq.Dequeue()

				So(err, ShouldBeNil)
				So(item, ShouldResemble, cr2)
				So(q.Len(), ShouldEqual, 2)
			})

			Convey("should take the least recently enqueued as third criteria", func() {
				component := testComponent("app-tq-1", "comp-tq-1", 1.0, 64.0)
				scheduled := scheduledComponent(&component)
				cr := &scheduled
				tq.Enqueue(cr)

				component2 := testComponent("app-tq-2", "comp-tq-2", 1.0, 64.0)
				scheduled2 := scheduledComponent(&component2)
				cr2 := &scheduled2
				tq.Enqueue(cr2)

				component3 := testComponent("app-tq-3", "comp-tq-3", 1.0, 64.0)
				scheduled3 := scheduledComponent(&component3)
				cr3 := &scheduled3
				tq.Enqueue(cr3)

				item, err := tq.Dequeue()

				So(err, ShouldBeNil)
				So(item, ShouldResemble, cr)
				So(q.Len(), ShouldEqual, 2)
			})
		})

		//Context("when being a persisted queue", func() {
		//PIt("should save the enqueued items to the underlying store", func() {})
		//PIt("should save the updates in orderting to the underlying store", func() {})
		//PIt("should initialize the task queue with the saved items on create", func() {})
		//PIt("should allow for stopping and releasing resources", func() {})
		//PIt("should allow for starting to acquire resources", func() {})
		//})
	})
}
