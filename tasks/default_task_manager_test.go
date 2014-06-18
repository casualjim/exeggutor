package tasks_test

import (
	"code.google.com/p/goprotobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	o "github.com/onsi/gomega"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	. "github.com/reverb/exeggutor/tasks"
	"github.com/reverb/go-mesos/mesos"
)

func testApp(appName string, cpus, mem float32) protocol.ApplicationManifest {
	comp := testComponent(appName, cpus, mem)
	return protocol.ApplicationManifest{
		Name:       proto.String(appName),
		Components: []*protocol.ApplicationComponent{&comp},
	}
}

func testComponent(compName string, cpus, mem float32) protocol.ApplicationComponent {
	distURL := "package://" + compName
	command := "./bin/" + compName
	version := "0.1.0"
	status := protocol.AppStatus_ABSENT
	scheme := "http"
	var port int32 = 8000
	logs := "./logs"
	work := "./work"
	conf := "./conf"
	dist := protocol.Distribution_PACKAGE
	comp := protocol.ComponentType_SERVICE
	return protocol.ApplicationComponent{
		Name:          proto.String(compName),
		Cpus:          proto.Float32(cpus),
		Mem:           proto.Float32(mem),
		DistUrl:       proto.String(distURL),
		Command:       proto.String(command),
		Version:       proto.String(version),
		Status:        &status,
		LogDir:        proto.String(logs),
		WorkDir:       proto.String(work),
		ConfDir:       proto.String(conf),
		Distribution:  &dist,
		ComponentType: &comp,
		Env:           []*protocol.StringKeyValue{},
		Ports: []*protocol.StringIntKeyValue{
			&protocol.StringIntKeyValue{
				Key:   proto.String(scheme),
				Value: proto.Int32(port),
			},
		},
	}
}

func scheduledComponent(appName string, component *protocol.ApplicationComponent) protocol.ScheduledAppComponent {
	return protocol.ScheduledAppComponent{
		Name:      component.Name,
		AppName:   proto.String(appName),
		Component: component,
		Position:  proto.Int(0),
		Since:     proto.Int64(5),
	}
}

func createOffer(id string, cpus, mem float64) mesos.Offer {
	offerID := &mesos.OfferID{
		Value: proto.String(id),
	}

	return mesos.Offer{
		Id: offerID,
		FrameworkId: &mesos.FrameworkID{
			Value: proto.String("exeggutor-tests-tasks-framework"),
		},
		SlaveId: &mesos.SlaveID{
			Value: proto.String("exeggutor-tests-tasks-slave"),
		},
		Hostname: proto.String("exeggutor-slave-instance-1"),
		Resources: []*mesos.Resource{
			mesos.ScalarResource("cpus", cpus),
			mesos.ScalarResource("mem", mem),
		},
		Attributes:  []*mesos.Attribute{},
		ExecutorIds: []*mesos.ExecutorID{},
	}
}

var _ = Describe("TaskManager", func() {
	var (
		mgr TaskManager
		q   *PrioQueue
		tq  TaskQueue
		ts  store.KVStore
		de  map[string]*mesos.TaskInfo
	)

	BeforeEach(func() {
		q = &PrioQueue{}
		tq = NewTaskQueueWithPrioQueue(q)
		tq.Start()
		ts = store.NewEmptyInMemoryStore()
		de = make(map[string]*mesos.TaskInfo)
		m, _ := NewCustomDefaultTaskManager(tq, ts, nil, de)
		m.Start()
		mgr = m
	})

	AfterEach(func() {
		tq.Stop()
		mgr.Stop()
	})

	Context("when enqueueing app manifests", func() {
		It("should enqueue an application manifest", func() {
			expected := testApp("test-service-1", 1.0, 256.0)
			err := mgr.SubmitApp(expected)

			Expect(err).NotTo(HaveOccurred())
			Expect(q.Len()).To(Equal(1))
			scheduled := scheduledComponent(expected.GetName(), expected.Components[0])
			actual := (*q)[0]
			actual.Since = proto.Int64(5)
			Expect(actual).To(Equal(&scheduled))
		})

		It("should enqueue all the components in a manifest", func() {
			expected := testApp("test-service-1", 1.0, 256.0)
			comp := testComponent("component-2", 1.0, 256.0)
			expected.Components = append(expected.Components, &comp)
			err := mgr.SubmitApp(expected)

			Expect(err).NotTo(HaveOccurred())
			Expect(q.Len()).To(Equal(2))

			var components []*protocol.ScheduledAppComponent
			for _, comp := range *q {
				comp.Since = proto.Int64(5)
				components = append(components, comp)
			}
			var expectedComponents []*protocol.ScheduledAppComponent
			for i, comp := range expected.Components {
				s := scheduledComponent(expected.GetName(), comp)
				scheduled := &s
				scheduled.Since = proto.Int64(5)
				scheduled.Position = proto.Int(i)
				expectedComponents = append(expectedComponents, scheduled)
			}
			Expect(components).To(Equal(expectedComponents))
		})

	})

	Context("when fullfilling offers", func() {

		It("should return an empty array when there are no apps queued", func() {
			offer := createOffer("offer-id-big", 5.0, 1024.0)
			res := mgr.FulfillOffer(offer)
			Expect(res).To(BeEmpty())
		})

		It("should fullfill an offer when there is an app queued that can statisfy it", func() {
			manifest := testApp("test-service-yada", 1.0, 256.0)
			component := manifest.Components[0]
			expectedCommand := BuildMesosCommand(component)
			expectedResources := BuildResources(component)
			mgr.SubmitApp(manifest)
			offer := createOffer("offer-id-1", 5.0, 1024.0)
			reply := mgr.FulfillOffer(offer)

			Expect(reply).NotTo(BeEmpty())
			Expect(reply).To(HaveLen(1))
			actual := reply[0]
			Expect(actual.Command).To(Equal(expectedCommand))
			Expect(actual.Resources).To(Equal(expectedResources))
		})

		It("should return an empty array when the offer can't be fullfilled", func() {
			manifest := testApp("test-service-yada", 5.0, 1024.0)

			mgr.SubmitApp(manifest)
			offer := createOffer("offer-id-1", 1.0, 256.0)
			reply := mgr.FulfillOffer(offer)

			Expect(reply).To(BeEmpty())
		})
	})

	Context("when handling callbacks", func() {

		It("should remove undeployed items from the deploying store when they fail", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			de[id.GetValue()] = &task
			mgr.TaskFailed(id, nil)
			Expect(de).NotTo(o.HaveKey(id.GetValue()))
		})

		It("should remove persisted items from the persistent store when they fail", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			bytes, _ := proto.Marshal(&task)
			ts.Set(id.GetValue(), bytes)
			mgr.TaskFailed(id, nil)
			actual, _ := ts.Get(id.GetValue())
			Expect(actual).To(BeNil())
		})

		It("should remove undeployed items from the deploying store when they finish", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			de[id.GetValue()] = &task
			mgr.TaskFinished(id, nil)
			Expect(de).NotTo(o.HaveKey(id.GetValue()))
		})

		It("should remove persisted items from the persistent store when they finish", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			bytes, _ := proto.Marshal(&task)
			ts.Set(id.GetValue(), bytes)
			mgr.TaskFinished(id, nil)
			actual, _ := ts.Get(id.GetValue())
			Expect(actual).To(BeNil())
		})

		It("should remove undeployed items from the deploying store when they are killed", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			de[id.GetValue()] = &task
			mgr.TaskKilled(id, nil)
			Expect(de).NotTo(o.HaveKey(id.GetValue()))
		})

		It("should remove persisted items from the persistent store when they are killed", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			bytes, _ := proto.Marshal(&task)
			ts.Set(id.GetValue(), bytes)
			mgr.TaskKilled(id, nil)
			actual, _ := ts.Get(id.GetValue())
			Expect(actual).To(BeNil())
		})

		It("should remove undeployed items from the deploying store when they are lost", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			de[id.GetValue()] = &task
			mgr.TaskLost(id, nil)
			Expect(de).NotTo(o.HaveKey(id.GetValue()))
		})

		It("should remove persisted items from the persistent store when they are lost", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			bytes, _ := proto.Marshal(&task)
			ts.Set(id.GetValue(), bytes)
			mgr.TaskLost(id, nil)
			actual, _ := ts.Get(id.GetValue())
			Expect(actual).To(BeNil())
		})

		It("should add to the persistence store if it exists in the deploying store for running", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			cr := &task
			de[id.GetValue()] = cr

			notThere, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(notThere).To(BeNil())
			mgr.TaskRunning(id, nil)

			bytes, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes).NotTo(BeNil())

			actual := mesos.TaskInfo{}
			proto.Unmarshal(bytes, &actual)
			Expect(actual).To(Equal(task))
		})

		It("shouldn't add to the persistence store when the item isn't in the deploying store", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()

			mgr.TaskRunning(id, nil)

			notThere, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(notThere).To(BeNil())

		})

		It("should add to the persistence store if it exists in the deploying store for staging", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			cr := &task
			de[id.GetValue()] = cr

			notThere, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(notThere).To(BeNil())
			mgr.TaskStaging(id, nil)

			bytes, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes).NotTo(BeNil())

			actual := mesos.TaskInfo{}
			proto.Unmarshal(bytes, &actual)
			Expect(actual).To(Equal(task))
		})

		It("shouldn't add to the persistence store when the item isn't in the deploying store for staging", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()

			mgr.TaskStaging(id, nil)

			notThere, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(notThere).To(BeNil())

		})

		It("should add to the persistence store if it exists in the deploying store for starting", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()
			cr := &task
			de[id.GetValue()] = cr

			notThere, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(notThere).To(BeNil())
			mgr.TaskStarting(id, nil)

			bytes, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes).NotTo(BeNil())

			actual := mesos.TaskInfo{}
			proto.Unmarshal(bytes, &actual)
			Expect(actual).To(Equal(task))
		})

		It("shouldn't add to the persistence store when the item isn't in the deploying store for starting", func() {
			offer := createOffer("offer id", 1.0, 64.0)
			component := testComponent("component name", 1.0, 64.0)
			scheduled := scheduledComponent("app name", &component)
			task := BuildTaskInfo("task id", &offer, &scheduled)
			id := task.GetTaskId()

			mgr.TaskStarting(id, nil)

			notThere, err := ts.Get(id.GetValue())
			Expect(err).NotTo(HaveOccurred())
			Expect(notThere).To(BeNil())

		})
	})
})
