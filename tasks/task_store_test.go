// Package tasks provides ...
package tasks

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"code.google.com/p/goprotobuf/proto"

	o "github.com/onsi/gomega"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
	. "github.com/smartystreets/goconvey/convey"
)

func createMulti(backing store.KVStore) []protocol.DeployedAppComponent {
	app1 := buildStoreTestData(1)
	app2 := buildStoreTestData(2)
	app3 := buildStoreTestData(3)
	saveStoreTestData(backing, &app1)
	saveStoreTestData(backing, &app2)
	saveStoreTestData(backing, &app3)
	return []protocol.DeployedAppComponent{app1, app2, app3}
}

func buildStoreTestData(index int) protocol.DeployedAppComponent {
	component := testComponent("app-store-"+strconv.Itoa(index), "app-"+strconv.Itoa(index), 1, 64)
	scheduled := scheduledComponent(&component)
	offer := createOffer("slave-"+strconv.Itoa(index), 8, 1024)
	task := BuildTaskInfo("task-app-id-"+strconv.Itoa(index), &offer, &scheduled)
	return deployedApp(&component, &task)
}

func saveStoreTestData(backing store.KVStore, deployed *protocol.DeployedAppComponent) {

	bytes, _ := proto.Marshal(deployed)
	backing.Set(deployed.TaskId.GetValue(), bytes)

}

func createStoreTestData(backing store.KVStore) (*mesos.TaskID, protocol.DeployedAppComponent) {

	deployed := buildStoreTestData(1)
	saveStoreTestData(backing, &deployed)
	return deployed.TaskId, deployed
}

const (
	success                             = ""
	needExactValues                     = "This assertion requires exactly %d comparison values (you provided %d)."
	shouldHaveProvidedCollectionMembers = "This assertion requires at least 1 comparison value (you provided 0)."
	shouldHaveBeenAValidCollection      = "You must provide a valid container (was %v)!"
	shouldHaveContained                 = "Expected the container (%v) to contain: '%v' (but it didn't)!"
)

func need(needed int, expected []interface{}) string {
	if len(expected) != needed {
		return fmt.Sprintf(needExactValues, needed, len(expected))
	}
	return success
}

func atLeast(minimum int, expected []interface{}) string {
	if len(expected) < 1 {
		return shouldHaveProvidedCollectionMembers
	}
	return success
}

func ShouldBeEquivalent(actual interface{}, expected ...interface{}) string {
	if fail := need(1, expected); fail != success {
		return fail
	}

	success, err := o.BeEquivalentTo(expected[0]).Match(actual)
	if err != nil {
		return err.Error()
	}
	if !success {
		return fmt.Sprintf("The collection %v did not match %v", expected[0], actual)
	}
	return ""

}

func ShouldHaveTheSameElementsAs(actual interface{}, expected ...interface{}) string {
	if fail := need(1, expected); fail != success {
		return fail
	}

	v1 := reflect.ValueOf(actual)
	if v1.Kind() != reflect.Slice && v1.Kind() != reflect.Array {
		return fmt.Sprintf(shouldHaveBeenAValidCollection, v1.Kind())
	}
	v2 := reflect.ValueOf(expected[0])
	if v2.Kind() != reflect.Slice && v2.Kind() != reflect.Array {
		return fmt.Sprintf(shouldHaveBeenAValidCollection, v2.Kind())
	}

	if v1.Len() != v2.Len() {
		return fmt.Sprintf("Expected actual to have %v items but it only has %v items", v1.Len(), v2.Len())
	}
	for i := 0; i < v2.Len(); i++ {
		found := false
		for j := 0; j < v1.Len(); j++ {
			if reflect.DeepEqual(v2.Index(i).Interface(), v1.Index(j).Interface()) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Sprintf(shouldHaveContained, actual, v2.Index(i).Interface())
		}
	}
	return ""
}

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
