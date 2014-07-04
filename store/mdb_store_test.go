package store

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMdbStore(t *testing.T) {

	Convey("MdbStore", t, func() {

		// Create a test dir
		tempDir, err := ioutil.TempDir("", "agora-store")
		So(err, ShouldBeNil)

		store, err := NewMdbStore(tempDir)
		So(err, ShouldBeNil)
		store.Start()

		context := DefaultExampleContext()
		for k, v := range context.Backing {
			store.Set(k, v)
		}
		context.Store = store

		Reset(func() {
			store.Stop()
			os.RemoveAll(tempDir)
		})

		SharedStoreBehavior(&context)

	})
}
