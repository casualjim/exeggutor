package store

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryStore(t *testing.T) {

	Convey("InMemoryStore", t, func() {

		context := DefaultExampleContext()
		store := NewInMemoryStore(context.Backing)
		context.Store = store

		Reset(func() {
			store.Stop()
		})

		SharedStoreBehavior(&context)

	})
}
