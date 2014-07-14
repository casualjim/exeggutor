// Package tasks provides ...
package tasks

import (
	"testing"

	"code.google.com/p/goprotobuf/proto"

	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/exeggutor/tasks/builders"
	. "github.com/reverb/exeggutor/test_utils"
	"github.com/reverb/go-mesos/mesos"
	. "github.com/reverb/go-utils/convey/matchers"
	"github.com/reverb/go-utils/flake"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTaskStore(t *testing.T) {

	context := &exeggutor.AppContext{
		Config: &exeggutor.Config{
			Mode: "test",
			DockerIndex: &exeggutor.DockerIndexConfig{
				Host: "dev-docker.helloreverb.com",
				Port: 443,
			},
		},
		IDGenerator: flake.NewFlake(),
	}

	Convey("A DefaultTaskStore", t, func() {

		builder := builders.New(context.Config)

		backing := store.NewEmptyInMemoryStore()
		taskStore := &DefaultTaskStore{store: backing}
		err := taskStore.Start()
		So(err, ShouldBeNil)

		Reset(func() {
			taskStore.Stop()
		})

		Convey("should get an item by key", func() {
			id, deployed := CreateStoreTestData(backing, builder)
			actual, err := taskStore.Get(id.GetValue())
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &deployed)
		})

		Convey("should save a deployed app", func() {
			id, deployed := CreateStoreTestData(backing, builder)
			bytes, _ := proto.Marshal(&deployed)

			err := taskStore.Save(&deployed)
			So(err, ShouldBeNil)
			retr, _ := backing.Get(id.GetValue())
			So(retr, ShouldResemble, bytes)
		})

		Convey("Should get the size", func() {
			CreateStoreTestData(backing, builder)
			sz, err := taskStore.Size()
			So(err, ShouldBeNil)
			So(sz, ShouldEqual, 1)
		})

		Convey("should delete an app", func() {
			id, _ := CreateStoreTestData(backing, builder)
			err := taskStore.Delete(id.GetValue())
			So(err, ShouldBeNil)
			contains, _ := backing.Contains(id.GetValue())
			So(contains, ShouldBeFalse)
		})

		Convey("should answer contains", func() {
			id, _ := CreateStoreTestData(backing, builder)
			contains, err := taskStore.Contains(id.GetValue())

			So(err, ShouldBeNil)
			So(contains, ShouldBeTrue)

			co2, err2 := taskStore.Contains("blah")

			So(err2, ShouldBeNil)
			So(co2, ShouldBeFalse)
		})

		Convey("should get the keys", func() {
			apps := CreateMulti(backing, builder)
			ids, err := taskStore.Keys()
			So(err, ShouldBeNil)
			So(ids, ShouldHaveTheSameElementsAs, []string{apps[0].TaskId.GetValue(), apps[1].TaskId.GetValue(), apps[2].TaskId.GetValue()})
		})

		Convey("should iterate over each value", func() {
			var expected []*protocol.Deployment
			for _, app := range CreateMulti(backing, builder) {
				expected = append(expected, &app)
			}
			var actual []*protocol.Deployment
			taskStore.ForEach(func(item *protocol.Deployment) {
				actual = append(actual, item)
			})

			So(actual, ShouldHaveTheSameElementsAs, expected)
		})

		Convey("should filter for task ids", func() {
			dd := CreateMulti(backing, builder)
			actual, err := taskStore.FilterToTaskIds(func(item *protocol.Deployment) bool {
				return item.GetAppId() == dd[1].GetAppId()
			})
			So(err, ShouldBeNil)
			So(actual, ShouldHaveTheSameElementsAs, []*mesos.TaskID{dd[1].TaskId})
		})

		Convey("should find an app", func() {
			dd := CreateMulti(backing, builder)
			actual, err := taskStore.Find(func(item *protocol.Deployment) bool {
				return item.GetAppId() == dd[1].GetAppId()
			})
			expected := dd[1]
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &expected)
		})
	})

}
