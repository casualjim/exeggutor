package queue

import (
	"fmt"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

type stringSerializer struct{}

func (s *stringSerializer) ReadBytes(data []byte) (interface{}, error) {
	return string(data), nil
}
func (s *stringSerializer) WriteBytes(target interface{}) ([]byte, error) {
	return []byte(fmt.Sprintf("%v", target)), nil
}

func newTestMdbQueue() *SharedQueueContext {
	// Create a test dir
	dir, err := ioutil.TempDir("", "agora-queue")
	if err != nil {
		Fail(fmt.Sprintf("err: %v ", err))
	}

	return &SharedQueueContext{
		Dir: dir,
		Factory: func() Queue {
			st, err := NewMdbQueue(dir, &stringSerializer{})
			if err != nil {
				Fail(err.Error())
			}
			return st
		},
	}
}

var _ = Describe("A persistent MdbQueue", func() {

	SharedQueueBehavior(newTestMdbQueue())

})
