// Package tasks provides ...
package store

import (
	"testing"

	"code.google.com/p/goprotobuf/proto"

	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	. "github.com/reverb/exeggutor/tasks"
	"github.com/reverb/go-mesos/mesos"
	. "github.com/reverb/go-utils/convey/matchers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTaskStore(t *testing.T) {

	Convey("A DefaultTaskStore", t, func() {
		backing := store.NewEmptyInMemoryStore()
		taskStore := &DefaultTaskStore{store: backing}
		err := taskStore.Start()
		So(err, ShouldBeNil)

		Reset(func() {
			taskStore.Stop()
		})

		Convey("should get an item by key", func() {
			id, deployed := createStoreTestData(backing)
			actual, err := taskStore.Get(id.GetValue())
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &deployed)
		})

		Convey("should save a deployed app", func() {
			component := testComponent("app-store-1", "app-1", 1, 64)
			scheduled := scheduledComponent(&component)
			offer := createOffer("slave-1", 8, 1024)
			task := BuildTaskInfo("task-app-id-1", &offer, &scheduled)
			deployed := deployedApp(&component, &task)

			bytes, _ := proto.Marshal(&deployed)

			err := taskStore.Save(&deployed)
			So(err, ShouldBeNil)
			retr, _ := backing.Get(task.TaskId.GetValue())
			So(retr, ShouldResemble, bytes)
		})

		Convey("Should get the size", func() {
			createStoreTestData(backing)
			sz, err := taskStore.Size()
			So(err, ShouldBeNil)
			So(sz, ShouldEqual, 1)
		})

		Convey("should delete an app", func() {
			id, _ := createStoreTestData(backing)
			err := taskStore.Delete(id.GetValue())
			So(err, ShouldBeNil)
			contains, _ := backing.Contains(id.GetValue())
			So(contains, ShouldBeFalse)
		})

		Convey("should answer contains", func() {
			id, _ := createStoreTestData(backing)
			contains, err := taskStore.Contains(id.GetValue())

			So(err, ShouldBeNil)
			So(contains, ShouldBeTrue)

			co2, err2 := taskStore.Contains("blah")

			So(err2, ShouldBeNil)
			So(co2, ShouldBeFalse)
		})

		Convey("should get the keys", func() {
			apps := createMulti(backing)
			ids, err := taskStore.Keys()
			So(err, ShouldBeNil)
			So(ids, ShouldHaveTheSameElementsAs, []string{apps[0].TaskId.GetValue(), apps[1].TaskId.GetValue(), apps[2].TaskId.GetValue()})
		})

		Convey("should iterate over each value", func() {
			var expected []*protocol.DeployedAppComponent
			for _, app := range createMulti(backing) {
				expected = append(expected, &app)
			}
			var actual []*protocol.DeployedAppComponent
			taskStore.ForEach(func(item *protocol.DeployedAppComponent) {
				actual = append(actual, item)
			})

			So(actual, ShouldHaveTheSameElementsAs, expected)
		})

		Convey("should filter for task ids", func() {
			dd := createMulti(backing)
			actual, err := taskStore.FilterToTaskIds(func(item *protocol.DeployedAppComponent) bool {
				return item.GetAppName() == dd[1].GetAppName()
			})
			So(err, ShouldBeNil)
			So(actual, ShouldHaveTheSameElementsAs, []*mesos.TaskID{dd[1].TaskId})
		})

		Convey("should find an app", func() {
			dd := createMulti(backing)
			actual, err := taskStore.Find(func(item *protocol.DeployedAppComponent) bool {
				return item.GetAppName() == dd[1].GetAppName()
			})
			expected := dd[1]
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &expected)
		})
	})

}
