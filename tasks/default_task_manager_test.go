package tasks

import (
	"code.google.com/p/goprotobuf/proto"
	. "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
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

var _ = Describe("TaskManager", func() {
	var (
		mgr TaskManager
		q   *prioQueue
		tq  TaskQueue
		ts  store.KVStore
	)

	BeforeEach(func() {
		q = &prioQueue{}
		tq = NewTaskQueueWithprioQueue(q)
		tq.Start()
		ts = store.NewEmptyInMemoryStore()
		m, _ := NewCustomDefaultTaskManager(tq, &DefaultTaskStore{ts}, nil)
		m.Start()
		mgr = m
	})

	AfterEach(func() {
		tq.Stop()
		mgr.Stop()
	})

	Context("when enqueueing app manifests", func() {
		It("should enqueue an application manifest", func() {
			expected := testComponent("test-service-1", "test-service-1", 1.0, 256.0)
			err := mgr.SubmitApp([]protocol.Application{expected})

			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(q.Len()).To(o.Equal(1))
			scheduled := scheduledComponent(&expected)
			actual := (*q)[0]
			actual.Since = proto.Int64(5)
			o.Expect(actual).To(o.Equal(&scheduled))
		})

		It("should enqueue all the components in a manifest", func() {
			expected := testComponent("test-service-1", "test-service-1", 1.0, 256.0)
			comp := testComponent("test-service-1", "component-2", 1.0, 256.0)
			app := []protocol.Application{expected, comp}
			err := mgr.SubmitApp(app)

			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(q.Len()).To(o.Equal(2))

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
			o.Expect(components).To(o.Equal(expectedComponents))
		})

	})

	Context("when fullfilling offers", func() {

		It("should return an empty array when there are no apps queued", func() {
			offer := createOffer("offer-id-big", 5.0, 1024.0)
			res := mgr.FulfillOffer(offer)
			o.Expect(res).To(o.BeEmpty())
		})

		It("should fullfill an offer when there is an app queued that can statisfy it", func() {
			component := testComponent("test-service-yada", "test-service-yada", 1.0, 256.0)
			expectedCommand := BuildMesosCommand("", &component)
			expectedResources := BuildResources(&component, []portRange{})
			mgr.SubmitApp([]protocol.Application{component})
			offer := createOffer("offer-id-1", 5.0, 1024.0)
			reply := mgr.FulfillOffer(offer)

			o.Expect(reply).NotTo(o.BeEmpty())
			o.Expect(reply).To(o.HaveLen(1))
			actual := reply[0]
			o.Expect(actual.Command).To(o.Equal(expectedCommand))
			o.Expect(actual.Resources[0]).To(o.Equal(expectedResources[0]))
			o.Expect(actual.Resources[1]).To(o.Equal(expectedResources[1]))
		})

		It("should return an empty array when the offer can't be fullfilled", func() {
			component := testComponent("test-service-yada", "test-service-yada", 5.0, 1024.0)

			mgr.SubmitApp([]protocol.Application{component})
			offer := createOffer("offer-id-1", 1.0, 256.0)
			reply := mgr.FulfillOffer(offer)

			o.Expect(reply).To(o.BeEmpty())
		})
	})

	Context("when handling callbacks", func() {

		It("should remove persisted items from the persistent store when they fail", func() {
			id, _ := setupCallbackTestData(ts)
			mgr.TaskFailed(id, nil)
			actual, _ := ts.Get(id.GetValue())
			o.Expect(actual).To(o.BeNil())
		})

		It("should remove persisted items from the persistent store when they finish", func() {
			id, _ := setupCallbackTestData(ts)
			mgr.TaskFinished(id, nil)
			actual, _ := ts.Get(id.GetValue())
			o.Expect(actual).To(o.BeNil())
		})

		It("should remove persisted items from the persistent store when they are killed", func() {
			id, _ := setupCallbackTestData(ts)
			mgr.TaskKilled(id, nil)
			actual, _ := ts.Get(id.GetValue())
			o.Expect(actual).To(o.BeNil())
		})

		It("should remove persisted items from the persistent store when they are lost", func() {
			id, _ := setupCallbackTestData(ts)

			mgr.TaskLost(id, nil)
			actual, _ := ts.Get(id.GetValue())
			o.Expect(actual).To(o.BeNil())
		})

		It("should add to the persistence store if it exists in the deploying store for running", func() {
			id, deployed := setupCallbackTestData(ts)
			mgr.TaskRunning(id, nil)

			bytes, err := ts.Get(id.GetValue())
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(bytes).NotTo(o.BeNil())
			ts.Set(id.GetValue(), bytes)

			actual := protocol.DeployedAppComponent{}
			proto.Unmarshal(bytes, &actual)
			deployed.Status = protocol.AppStatus_STARTED.Enum()
			o.Expect(actual).To(o.Equal(deployed))
		})

		It("should remove persisted items from the store for staging", func() {
			id, _ := setupCallbackTestData(ts)

			mgr.TaskStaging(id, nil)
			actual, _ := ts.Get(id.GetValue())
			o.Expect(actual).To(o.BeNil())
		})

		It("should remove persisted items from the store for starting", func() {
			id, _ := setupCallbackTestData(ts)

			mgr.TaskStarting(id, nil)

			actual, _ := ts.Get(id.GetValue())
			o.Expect(actual).To(o.BeNil())

		})
	})
})
