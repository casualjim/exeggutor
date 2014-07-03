// Package tasks_test provides ...
package tasks

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/reverb/exeggutor/protocol"
)

var _ = Describe("A DefaultTaskQueue", func() {
	var (
		q  *prioQueue
		tq TaskQueue
	)

	BeforeEach(func() {
		q = &prioQueue{}
		tq = NewTaskQueueWithprioQueue(q)
		tq.Start()
	})

	AfterEach(func() {
		tq.Stop()
	})

	Context("when behaving as a queue", func() {

		It("should allow putting things onto the queue", func() {
			component := testComponent("app-tq-1", "comp-tq-1", 1.0, 64.0)
			scheduled := scheduledComponent(&component)
			tq.Enqueue(&scheduled)
		})

		It("should allow popping things of the queue", func() {
			component := testComponent("app-tq-1", "comp-tq-1", 1.0, 64.0)
			scheduled := scheduledComponent(&component)
			cr := &scheduled
			tq.Enqueue(cr)

			component2 := testComponent("app-tq-2", "comp-tq-2", 1.0, 64.0)
			scheduled2 := scheduledComponent(&component2)
			cr2 := &scheduled2
			tq.Enqueue(cr2)

			item, err := tq.Dequeue()

			Expect(err).NotTo(HaveOccurred())
			Expect(item).To(Equal(cr))
			Expect(item).NotTo(Equal(cr2))
			Expect(q.Len()).To(Equal(1))
		})

		It("should allow picking the first thing that matches a predicate for dequeue", func() {
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

			Expect(err).NotTo(HaveOccurred())
			Expect(item).NotTo(Equal(cr))
			Expect(item).To(Equal(cr2))
			Expect(q.Len()).To(Equal(1))
		})
	})

	Context("when being a priority queue", func() {

		It("should take the thing with the largest cpu needs first", func() {
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

			Expect(err).NotTo(HaveOccurred())
			Expect(item).To(Equal(cr3))
			Expect(q.Len()).To(Equal(2))
		})

		It("should take the thing with the largest memory needs second", func() {
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

			Expect(err).NotTo(HaveOccurred())
			Expect(item).To(Equal(cr2))
			Expect(q.Len()).To(Equal(2))
		})

		It("should take the least recently enqueued as third criteria", func() {
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

			Expect(err).NotTo(HaveOccurred())
			Expect(item).To(Equal(cr))
			Expect(q.Len()).To(Equal(2))
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
