// Package sla provides ...
package sla

import (
	stdlog "log"
	"os"
	"testing"
	"time"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	app_store "github.com/reverb/exeggutor/store/apps"
	task_store "github.com/reverb/exeggutor/store/tasks"
	"github.com/reverb/exeggutor/tasks/builders"
	task_queue "github.com/reverb/exeggutor/tasks/queue"
	"github.com/reverb/exeggutor/test_utils"
	"github.com/reverb/go-utils/flake"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSLAMonitorStartupLoop(t *testing.T) {

	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logging.SetBackend(logBackend)
	logging.SetLevel(logging.DEBUG, "")

	context := &exeggutor.AppContext{
		Config: &exeggutor.Config{
			Mode: "test",
			DockerIndex: &exeggutor.DockerIndexConfig{
				Host: "dev-docker.helloreverb.com",
				Port: 443,
			},
			FrameworkInfo: &exeggutor.FrameworkConfig{
				HealthCheckConcurrency: 1,
			},
		},
		IDGenerator: flake.NewFlake(),
	}

	Convey("A SLA Monitor", t, func() {
		q := &task_queue.PrioQueue{}
		tq := task_queue.NewTaskQueueWithPrioQueue(q)
		tq.Start()
		as := store.NewEmptyInMemoryStore()
		ts := store.NewEmptyInMemoryStore()
		taskStore := task_store.NewWithStore(ts)
		appStore := app_store.NewWithStore(as)
		builder := builders.New(context.Config)
		monitor := NewWithInterval(taskStore, appStore, tq, 1*time.Second)
		monitor.Start()

		Reset(func() {
			monitor.Stop()
			tq.Stop()
		})

		Convey("it should start without hanging", func() {
			AppWithSla(context, 1, 1, 1)
			So(true, ShouldBeTrue)
		})

		Convey("it should raise an increase event eventually", func() {
			deployment, component := AppWithSla(context, 1, 3, 4)
			scheduled := test_utils.ScheduledComponent(&component)
			tq.Enqueue(&scheduled)
			taskStore.Save(&deployment)
			appStore.Save(&component)
			cnt := <-monitor.ScaleUpOrDown()
			So(cnt.Count, ShouldEqual, 1)
		})

		Convey("it should raise an increase event eventually", func() {
			deployment, component := AppWithSla(context, 1, 1, 1)
			deployment.Status = protocol.AppStatus_STARTED.Enum()
			taskStore.Save(&deployment)

			offer := test_utils.CreateOffer("offer-4949", 1, 1024)
			scheduled := test_utils.ScheduledComponent(&component)
			task, _ := builder.BuildTaskInfo("task-94994", &offer, &scheduled)
			deployed := test_utils.DeployedApp(&component, &task)
			taskStore.Save(&deployed)
			appStore.Save(&component)
			cnt := <-monitor.ScaleUpOrDown()
			So(cnt.Count, ShouldEqual, -1)
		})
	})
}
