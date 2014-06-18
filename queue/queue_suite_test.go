package queue

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	pth := fmt.Sprintf("../test-reports/junit_exeggutor_queue_%d.xml", config.GinkgoConfig.ParallelNode)
	junitReporter := reporters.NewJUnitReporter(pth)
	RunSpecsWithDefaultAndCustomReporters(t, "Exeggutor Queue Test Suite", []Reporter{junitReporter})
}

type SharedQueueContext struct {
	Dir     string
	Factory func() Queue
}

func SharedQueueBehavior(context *SharedQueueContext) {
	var queue Queue

	BeforeEach(func() {
		queue = context.Factory()
		queue.Start()
	})

	AfterEach(func() {
		queue.Stop()
		if context.Dir != "" {
			os.RemoveAll(context.Dir)
		}
	})

	It("is empty on initialization", func() {
		Expect(queue.IsEmpty()).To(BeTrue())
	})

	It("has len == 0 on construction", func() {
		Expect(queue.Len()).To(Equal(0))
	})

	It("is not empty after 1 or more enqueues", func() {
		Expect(queue.IsEmpty()).To(BeTrue())
		noOfInserts := 6
		for i := 0; i < noOfInserts; i++ {
			err := queue.Enqueue("blah")
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(queue.IsEmpty()).To(BeFalse())
		Expect(queue.Len()).To(Equal(noOfInserts))
	})

	It("dequeues a previously enqueued value", func() {
		queue.Enqueue("hello")
		v, err := queue.Dequeue()
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal("hello"))
	})

	It("doesn't dequeue when peeked at the next value", func() {
		queue.Enqueue("blah3030")
		v, err := queue.Peek()
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal("blah3030"))
		Expect(queue.Len()).To(Equal(1))
	})

	It("is empty when all the values are dequeued", func() {
		noOfRemoves := rand.Intn(50)

		for i := 0; i < noOfRemoves; i++ {
			err := queue.Enqueue("blah3939")
			Expect(err).NotTo(HaveOccurred())
		}
		for i := 0; i < noOfRemoves; i++ {
			_, err := queue.Dequeue()
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(queue.IsEmpty()).To(BeTrue())
		Expect(queue.Len()).To(Equal(0))
	})

	It("deqeueues as the size of the queue, after which the queue is empty", func() {
		noOfInserts := 50
		for i := 0; i < noOfInserts; i++ {
			err := queue.Enqueue(strconv.Itoa(i))
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(queue.Len()).To(Equal(noOfInserts))
		for i := 0; i < noOfInserts; i++ {
			v, err := queue.Dequeue()
			Expect(err).NotTo(HaveOccurred())
			Expect(strconv.Atoi(v.(string))).To(Equal(i))
		}
		Expect(queue.IsEmpty()).To(BeTrue())
	})

	It("dequeueing an empty queue returns nil", func() {
		Expect(queue.IsEmpty()).To(BeTrue())
		v, err := queue.Dequeue()
		Expect(err).NotTo(HaveOccurred())
		Expect(queue.IsEmpty()).To(BeTrue())
		Expect(v).To(BeNil())
	})

	It("peeking an empty queue returns nil", func() {
		Expect(queue.IsEmpty()).To(BeTrue())
		v, err := queue.Peek()
		Expect(err).NotTo(HaveOccurred())
		Expect(queue.IsEmpty()).To(BeTrue())
		Expect(v).To(BeNil())
	})

	It("returns an error when trying to enqueue nil", func() {
		err := queue.Enqueue(nil)
		Expect(err).To(HaveOccurred())
	})
}
