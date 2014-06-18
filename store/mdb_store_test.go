package store

import (
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("MdbStore", func() {

	var (
		context StoreExampleContext = StoreExampleContext{}
		store   KVStore
		tempDir string
	)

	BeforeEach(func() {
		// Create a test dir
		dir, err := ioutil.TempDir("", "agora-store")
		if err != nil {
			Fail(fmt.Sprintf("err: %v ", err))
		}
		tempDir = dir

		st, err := NewMdbStore(dir)
		st.Start()
		if err != nil {
			Fail(fmt.Sprintf("err: %v ", err))
		}

		ct := DefaultExampleContext()
		for k, v := range ct.Backing {
			st.Set(k, v)
		}
		store = st
		context.Store = store
		context.Backing = ct.Backing
		context.Keys = ct.Keys
		context.Values = ct.Values
	})

	AfterEach(func() {
		store.Stop()
		os.RemoveAll(tempDir)
	})

	SharedStoreBehavior(&context)

})
