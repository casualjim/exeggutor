package queue

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("An InMemoryQueue", func() {

	SharedQueueBehavior(&SharedQueueContext{Dir: "", Factory: func() Queue { return NewInMemoryQueue() }})

})
