package tasks

import (
	"os"
	"testing"

	stdlog "log"

	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/exeggutor/tasks/builders"
	task_queue "github.com/reverb/exeggutor/tasks/queue"
	task_store "github.com/reverb/exeggutor/tasks/store"
	. "github.com/reverb/exeggutor/tasks/test_utils"
	"github.com/reverb/go-mesos/mesos"
	. "github.com/reverb/go-utils/convey/matchers"
	"github.com/reverb/go-utils/flake"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTaskManager(t *testing.T) {
	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true
	logging.SetBackend(logBackend)
	logging.SetLevel(logging.ERROR, "")

	context := &exeggutor.AppContext{
		Config: &exeggutor.Config{
			Mode:        "test",
			DockerIndex: "dev-docker.helloreverb.com",
		},
		IDGenerator: flake.NewFlake(),
	}

	Convey("TaskManager", t, func() {
		builder := builders.New(context.Config)
		builder.PortPicker = &ConstantPortPicker{Port: 8000}

		q := &task_queue.PrioQueue{}
		tq := task_queue.NewTaskQueueWithPrioQueue(q)
		tq.Start()
		ts := store.NewEmptyInMemoryStore()
		mgr := &DefaultTaskManager{
			queue:       tq,
			taskStore:   task_store.NewWithStore(ts),
			context:     context,
			builder:     builder,
			healtchecks: nil,
		}
		mgr.Start()

		Reset(func() {
			tq.Stop()
			mgr.Stop()
		})

		Convey("when enqueueing app manifests", func() {
			Convey("should enqueue an application manifest", func() {
				expected := TestComponent("test-service-1", "test-service-1", 1.0, 256.0)
				err := mgr.SubmitApp([]protocol.Application{expected})

				So(err, ShouldBeNil)
				So(q.Len(), ShouldEqual, 1)
				scheduled := ScheduledComponent(&expected)
				actual := (*q)[0]
				actual.Since = proto.Int64(5)
				So(actual, ShouldResemble, &scheduled)
			})

			Convey("should enqueue all the components in a manifest", func() {
				expected := TestComponent("test-service-1", "test-service-1", 1.0, 256.0)
				comp := TestComponent("test-service-1", "component-2", 1.0, 256.0)
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
					s := ScheduledComponent(&comp)
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
				offer := CreateOffer("offer-id-big", 5.0, 1024.0)
				res := mgr.FulfillOffer(offer)
				So(res, ShouldBeEmpty)
			})

			Convey("should fullfill an offer when there is an app queued that can statisfy it", func() {
				component := TestComponent("test-service-yada", "test-service-yada", 1.0, 256.0)
				prange, p := builders.PortRangeFor(8000)
				expectedCommand, _ := builder.BuildMesosCommand("", &component, p)
				expectedResources := builder.BuildResources(&component, prange)
				mgr.SubmitApp([]protocol.Application{component})
				offer := CreateOffer("offer-id-1", 5.0, 1024.0)
				reply := mgr.FulfillOffer(offer)

				So(reply, ShouldNotBeEmpty)
				So(len(reply), ShouldEqual, 1)
				actual := reply[0]
				So(actual.Command, ShouldResemble, expectedCommand)
				So(actual.Resources, ShouldResemble, expectedResources)
			})

			Convey("should return an empty array when the offer can't be fullfilled", func() {
				component := TestComponent("test-service-yada", "test-service-yada", 5.0, 1024.0)

				mgr.SubmitApp([]protocol.Application{component})
				offer := CreateOffer("offer-id-1", 1.0, 256.0)
				reply := mgr.FulfillOffer(offer)

				So(reply, ShouldBeEmpty)
			})
		})

		Convey("when handling callbacks", func() {

			Convey("should remove persisted items from the persistent store when they fail", func() {
				id, deployed := SetupCallbackTestData(ts, builder)
				mgr.TaskFailed(id, nil)

				bytes, err := ts.Get(id.GetValue())
				So(err, ShouldBeNil)
				So(bytes, ShouldNotBeNil)

				actual := protocol.DeployedAppComponent{}
				proto.Unmarshal(bytes, &actual)
				deployed.Status = protocol.AppStatus_STOPPED.Enum()
				So(actual, ShouldResemble, deployed)
			})

			Convey("should remove persisted items from the persistent store when they finish", func() {
				id, deployed := SetupCallbackTestData(ts, builder)
				mgr.TaskFinished(id, nil)

				bytes, err := ts.Get(id.GetValue())
				So(err, ShouldBeNil)
				So(bytes, ShouldNotBeNil)

				actual := protocol.DeployedAppComponent{}
				proto.Unmarshal(bytes, &actual)
				deployed.Status = protocol.AppStatus_STOPPED.Enum()
				So(actual, ShouldResemble, deployed)
			})

			Convey("should remove persisted items from the persistent store when they are killed", func() {
				id, deployed := SetupCallbackTestData(ts, builder)
				mgr.TaskKilled(id, nil)

				bytes, err := ts.Get(id.GetValue())
				So(err, ShouldBeNil)
				So(bytes, ShouldNotBeNil)

				actual := protocol.DeployedAppComponent{}
				proto.Unmarshal(bytes, &actual)
				deployed.Status = protocol.AppStatus_STOPPED.Enum()
				So(actual, ShouldResemble, deployed)
			})

			Convey("should remove persisted items from the persistent store when they are lost", func() {
				id, deployed := SetupCallbackTestData(ts, builder)
				mgr.TaskLost(id, nil)

				bytes, err := ts.Get(id.GetValue())
				So(err, ShouldBeNil)
				So(bytes, ShouldNotBeNil)

				actual := protocol.DeployedAppComponent{}
				proto.Unmarshal(bytes, &actual)
				deployed.Status = protocol.AppStatus_STOPPED.Enum()
				So(actual, ShouldResemble, deployed)
			})

			Convey("should add to the persistence store if it exists in the deploying store for running", func() {
				id, deployed := SetupCallbackTestData(ts, builder)
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
				id, _ := SetupCallbackTestData(ts, builder)

				mgr.TaskStaging(id, nil)
				actual, _ := ts.Get(id.GetValue())
				So(actual, ShouldBeNil)
			})

			Convey("should remove persisted items from the store for starting", func() {
				id, _ := SetupCallbackTestData(ts, builder)

				mgr.TaskStarting(id, nil)

				actual, _ := ts.Get(id.GetValue())
				So(actual, ShouldBeNil)

			})
		})

		Convey("when finding deployed apps", func() {
			Convey("should find all components for a specified app", func() {
				apps := CreateFilterData(ts, builder)
				expected := []*mesos.TaskID{apps[0].TaskId, apps[1].TaskId, apps[2].TaskId}

				actual, err := mgr.FindTasksForApp(apps[0].GetAppName())
				So(err, ShouldBeNil)
				So(actual, ShouldHaveTheSameElementsAs, expected)
			})

			Convey("should find all instances of a component for a specified app", func() {
				apps := CreateFilterData(ts, builder)
				expected := []*mesos.TaskID{apps[3].TaskId, apps[5].TaskId}

				actual, err := mgr.FindTasksForComponent(apps[3].GetAppName(), apps[3].Component.GetName())
				So(err, ShouldBeNil)
				So(actual, ShouldHaveTheSameElementsAs, expected)
			})

			Convey("should find a specific component instance", func() {
				apps := CreateFilterData(ts, builder)
				expected := apps[4].TaskId

				actual, err := mgr.FindTaskForComponent(apps[4].TaskId.GetValue())
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, expected)
			})
		})
	})
}
