package store_test

import (
	. "github.com/reverb/exeggutor/store"

	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("InMemoryStore", func() {

	var (
		context StoreExampleContext = StoreExampleContext{}
		store   KVStore
	)

	BeforeEach(func() {
		ct := DefaultExampleContext()
		store = NewInMemoryStore(ct.Backing)
		context.Store = store
		context.Backing = ct.Backing
		context.Keys = ct.Keys
		context.Values = ct.Values
	})

	AfterEach(func() {
		store.Stop()
	})

	SharedStoreBehavior(&context)

})
