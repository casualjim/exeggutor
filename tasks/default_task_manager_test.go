package tasks_test

import (
	"code.google.com/p/goprotobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/queue"
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
		q   queue.Queue
		ts  store.KVStore
	)

	BeforeEach(func() {
		q = queue.NewInMemoryQueue()
		ts = store.NewEmptyInMemoryStore()
		m, _ := NewCustomDefaultTaskManager(q, ts, nil)
		m.Start()
		mgr = m
	})

	AfterEach(func() {
		mgr.Stop()
	})

	Context("when enqueueing app manifests", func() {
		It("should enqueue an application manifest", func() {
			expected := testApp("test-service-1", 1.0, 256.0)
			err := mgr.SubmitApp(expected)

			Expect(err).NotTo(HaveOccurred())
			Expect(q.Len()).To(Equal(1))
			Expect(q.Peek()).To(Equal(scheduledComponent(expected.GetName(), expected.Components[0])))
		})

		It("should enqueue all the components in a manifest", func() {
			expected := testApp("test-service-1", 1.0, 256.0)
			comp := testComponent("component-2", 1.0, 256.0)
			expected.Components = append(expected.Components, &comp)
			err := mgr.SubmitApp(expected)

			Expect(err).NotTo(HaveOccurred())
			Expect(q.Len()).To(Equal(2))

			var components []protocol.ScheduledAppComponent
			q.ForEach(func(data interface{}) {
				components = append(components, data.(protocol.ScheduledAppComponent))
			})
			var expectedComponents []protocol.ScheduledAppComponent
			for _, comp := range expected.Components {
				expectedComponents = append(expectedComponents, scheduledComponent(expected.GetName(), comp))
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
})
