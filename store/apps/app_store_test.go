// Package apps provides ...
package apps

import (
	"testing"

	"code.google.com/p/goprotobuf/proto"

	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/exeggutor/tasks/builders"
	. "github.com/reverb/exeggutor/test_utils"
	. "github.com/reverb/go-utils/convey/matchers"
	"github.com/reverb/go-utils/flake"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAppStore(t *testing.T) {

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

	Convey("A DefaultAppStore", t, func() {

		builder := builders.New(context.Config)

		backing := store.NewEmptyInMemoryStore()
		appStore := &DefaultAppStore{store: backing}
		err := appStore.Start()
		So(err, ShouldBeNil)

		Reset(func() {
			appStore.Stop()
		})

		Convey("should get an item by key", func() {
			app := CreateAppStoreTestData(backing, builder)
			actual, err := appStore.Get(app.GetId())
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &app)
		})

		Convey("should save an app", func() {
			app := CreateAppStoreTestData(backing, builder)
			bytes, _ := proto.Marshal(&app)

			err := appStore.Save(&app)
			So(err, ShouldBeNil)
			retr, _ := backing.Get(app.GetId())
			So(retr, ShouldResemble, bytes)
		})

		Convey("Should get the size", func() {
			CreateAppStoreTestData(backing, builder)
			sz, err := appStore.Size()
			So(err, ShouldBeNil)
			So(sz, ShouldEqual, 1)
		})

		Convey("should delete an app", func() {
			app := CreateAppStoreTestData(backing, builder)
			err := appStore.Delete(app.GetId())
			So(err, ShouldBeNil)
			contains, _ := backing.Contains(app.GetId())
			So(contains, ShouldBeFalse)
		})

		Convey("should answer contains", func() {
			app := CreateAppStoreTestData(backing, builder)
			contains, err := appStore.Contains(app.GetId())

			So(err, ShouldBeNil)
			So(contains, ShouldBeTrue)

			co2, err2 := appStore.Contains("blah")

			So(err2, ShouldBeNil)
			So(co2, ShouldBeFalse)
		})

		Convey("should get the keys", func() {
			apps := CreateAppStoreMulti(backing, builder)
			ids, err := appStore.Keys()
			So(err, ShouldBeNil)
			So(ids, ShouldHaveTheSameElementsAs, []string{apps[0].GetId(), apps[1].GetId(), apps[2].GetId()})
		})

		Convey("should iterate over each value", func() {
			var expected []*protocol.Application
			for _, app := range CreateAppStoreMulti(backing, builder) {
				expected = append(expected, &app)
			}
			var actual []*protocol.Application
			appStore.ForEach(func(item *protocol.Application) {
				actual = append(actual, item)
			})

			So(actual, ShouldHaveTheSameElementsAs, expected)
		})

		Convey("should find an app", func() {
			dd := CreateAppStoreMulti(backing, builder)
			actual, err := appStore.Find(func(item *protocol.Application) bool {
				return item.GetAppName() == dd[1].GetAppName()
			})
			expected := dd[1]
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &expected)
		})
	})

}
