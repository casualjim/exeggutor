package queue_test

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
	. "github.com/reverb/exeggutor/queue"
)

var _ = Describe("An InMemoryQueue", func() {

	SharedQueueBehavior(&SharedQueueContext{Dir: "", Factory: func() Queue { return NewInMemoryQueue() }})

})
