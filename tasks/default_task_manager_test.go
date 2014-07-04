package tasks

import (
	"os"
	"testing"

	stdlog "log"

	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
	. "github.com/smartystreets/goconvey/convey"
)

func setupCallbackTestData(ts store.KVStore) (*mesos.TaskID, protocol.DeployedAppComponent) {
	offer := createOffer("offer id", 1.0, 64.0)
	component := testComponent("app name", "component name", 1.0, 64.0)
	cr := &component
	scheduled := scheduledComponent(cr)
	task := BuildTaskInfo("task id", &offer, &scheduled)
	tr := &task
	id := task.GetTaskId()
	deployed := deployedApp(cr, tr)
	bytes, _ := proto.Marshal(&deployed)
	ts.Set(id.GetValue(), bytes)

	return id, deployed
}

func TestTaskManager(t *testing.T) {
	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true
	logging.SetBackend(logBackend)
	logging.SetLevel(logging.ERROR, "")

	Convey("TaskManager", t, func() {
		q := &prioQueue{}
		tq := NewTaskQueueWithprioQueue(q)
		tq.Start()
		ts := store.NewEmptyInMemoryStore()
		mgr, _ := NewCustomDefaultTaskManager(tq, &DefaultTaskStore{ts}, nil)
		mgr.Start()

		Reset(func() {
			tq.Stop()
			mgr.Stop()
		})

		Convey("when enqueueing app manifests", func() {
			Convey("should enqueue an application manifest", func() {
				expected := testComponent("test-service-1", "test-service-1", 1.0, 256.0)
				err := mgr.SubmitApp([]protocol.Application{expected})

				So(err, ShouldBeNil)
				So(q.Len(), ShouldEqual, 1)
				scheduled := scheduledComponent(&expected)
				actual := (*q)[0]
				actual.Since = proto.Int64(5)
				So(actual, ShouldResemble, &scheduled)
			})

			Convey("should enqueue all the components in a manifest", func() {
				expected := testComponent("test-service-1", "test-service-1", 1.0, 256.0)
				comp := testComponent("test-service-1", "component-2", 1.0, 256.0)
				app := []protocol.Application{expected, comp}
				err := mgr.SubmitApp(app)

				So(err, ShouldBeNil)
				So(q.Len(), ShouldEqual, 2)

				var components []*protocol.ScheduledApp
				for _, comp := range *q {
					comp.Since = proto.Int64(5)
					components = append(components, comp)
				}
				var expectedComponents []*protocol.ScheduledApp
				for i, comp := range app {
					s := scheduledComponent(&comp)
					scheduled := &s
					scheduled.Since = proto.Int64(5)
					scheduled.Position = proto.Int(i)
					expectedComponents = append(expectedComponents, scheduled)
				}
				So(components, ShouldResemble, expectedComponents)
			})

		})

		Convey("when fullfilling offers", func() {

			Convey("should return an empty array when there are no apps queued", func() {
				offer := createOffer("offer-id-big", 5.0, 1024.0)
				res := mgr.FulfillOffer(offer)
				So(res, ShouldBeEmpty)
			})

			Convey("should fullfill an offer when there is an app queued that can statisfy it", func() {
				component := testComponent("test-service-yada", "test-service-yada", 1.0, 256.0)
				expectedCommand := BuildMesosCommand("", &component)
				expectedResources := BuildResources(&component, []portRange{})
				mgr.SubmitApp([]protocol.Application{component})
				offer := createOffer("offer-id-1", 5.0, 1024.0)
				reply := mgr.FulfillOffer(offer)

				So(reply, ShouldNotBeEmpty)
				So(len(reply), ShouldEqual, 1)
				actual := reply[0]
				So(actual.Command, ShouldResemble, expectedCommand)
				So(actual.Resources[0], ShouldResemble, expectedResources[0])
				So(actual.Resources[1], ShouldResemble, expectedResources[1])
			})

			Convey("should return an empty array when the offer can't be fullfilled", func() {
				component := testComponent("test-service-yada", "test-service-yada", 5.0, 1024.0)

				mgr.SubmitApp([]protocol.Application{component})
				offer := createOffer("offer-id-1", 1.0, 256.0)
				reply := mgr.FulfillOffer(offer)

				So(reply, ShouldBeEmpty)
			})
		})

		Convey("when handling callbacks", func() {

			Convey("should remove persisted items from the persistent store when they fail", func() {
				id, _ := setupCallbackTestData(ts)
				mgr.TaskFailed(id, nil)
				actual, _ := ts.Get(id.GetValue())
				So(actual, ShouldBeNil)
			})

			Convey("should remove persisted items from the persistent store when they finish", func() {
				id, _ := setupCallbackTestData(ts)
				mgr.TaskFinished(id, nil)
				actual, _ := ts.Get(id.GetValue())
				So(actual, ShouldBeNil)
			})

			Convey("should remove persisted items from the persistent store when they are killed", func() {
				id, _ := setupCallbackTestData(ts)
				mgr.TaskKilled(id, nil)
				actual, _ := ts.Get(id.GetValue())
				So(actual, ShouldBeNil)
			})

			Convey("should remove persisted items from the persistent store when they are lost", func() {
				id, _ := setupCallbackTestData(ts)

				mgr.TaskLost(id, nil)
				actual, _ := ts.Get(id.GetValue())
				So(actual, ShouldBeNil)
			})

			Convey("should add to the persistence store if it exists in the deploying store for running", func() {
				id, deployed := setupCallbackTestData(ts)
				mgr.TaskRunning(id, nil)

				bytes, err := ts.Get(id.GetValue())
				So(err, ShouldBeNil)
				So(bytes, ShouldNotBeNil)

				actual := protocol.DeployedAppComponent{}
				proto.Unmarshal(bytes, &actual)
				deployed.Status = protocol.AppStatus_STARTED.Enum()
				So(actual, ShouldResemble, deployed)
			})

			Convey("should remove persisted items from the store for staging", func() {
				id, _ := setupCallbackTestData(ts)

				mgr.TaskStaging(id, nil)
				actual, _ := ts.Get(id.GetValue())
				So(actual, ShouldBeNil)
			})

			Convey("should remove persisted items from the store for starting", func() {
				id, _ := setupCallbackTestData(ts)

				mgr.TaskStarting(id, nil)

				actual, _ := ts.Get(id.GetValue())
				So(actual, ShouldBeNil)

			})
		})

		// Convey("when finding deployed apps", func() {
		// 	PConvey("should find all components for a specified app", func() {

		// 	})
		// 	PConvey("should find all instances of a component for a specified app", func() {

		// 	})
		// 	PConvey("should find a specific component instance", func() {

		// 	})
		// })
	})
}
