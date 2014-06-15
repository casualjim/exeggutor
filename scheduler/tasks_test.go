package scheduler_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/queue"
	. "github.com/reverb/exeggutor/scheduler"
	"github.com/reverb/exeggutor/store"
)

func testApp(appName string) protocol.ApplicationManifest {
	var cpus float32 = 1.0
	var mem float32 = 256
	distUrl := "package://" + appName
	command := "./bin/" + appName
	version := "0.1.0"
	status := protocol.AppStatus_ABSENT
	scheme := "http"
	var port int32 = 8000
	logs := "./logs"
	work := "./work"
	conf := "./conf"
	dist := protocol.Distribution_PACKAGE
	comp := protocol.ComponentType_SERVICE

	return protocol.ApplicationManifest{
		Name: &appName,
		Components: []*protocol.ApplicationComponent{
			&protocol.ApplicationComponent{
				Name:          &appName,
				Cpus:          &cpus,
				Mem:           &mem,
				DistUrl:       &distUrl,
				Command:       &command,
				Version:       &version,
				Status:        &status,
				LogDir:        &logs,
				WorkDir:       &work,
				ConfDir:       &conf,
				Distribution:  &dist,
				ComponentType: &comp,
				Env:           []*protocol.StringKeyValue{},
				Ports: []*protocol.StringIntKeyValue{
					&protocol.StringIntKeyValue{
						Key:   &scheme,
						Value: &port,
					},
				},
			},
		},
	}
}

var _ = Describe("TaskManager", func() {
	var (
		mgr *TaskManager
		q   queue.Queue
		ts  store.KVStore
	)

	BeforeEach(func() {
		q = queue.NewInMemoryQueue()
		ts = store.NewEmptyInMemoryStore()
		m, _ := NewCustomTaskManager(q, ts, nil)
		m.Start()
		mgr = m
	})

	AfterEach(func() {
		mgr.Stop()
	})

	It("should enqueue an application manifest", func() {
		expected := testApp("test-service-1")
		err := mgr.SubmitApp(expected)

		Expect(err).NotTo(HaveOccurred())

		Expect(q.Len()).To(Equal(1))
		Expect(q.Peek()).To(Equal(expected))
	})
})
