// Package sla provides ...
package sla

import (
	stdlog "log"
	"os"
	"testing"
	"time"

	"code.google.com/p/goprotobuf/proto"

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

func AppWithSla(context *exeggutor.AppContext, index int, minInstances, maxInstances int32) (protocol.Deployment, protocol.Application) {
	deployment, app := test_utils.BuildStoreTestData(index, builders.New(context.Config))
	app.Sla = &protocol.ApplicationSLA{
		MinInstances: proto.Int32(minInstances),
		MaxInstances: proto.Int32(maxInstances),
		HealthCheck: &protocol.HealthCheck{
			Mode:           protocol.HealthCheckMode_HTTP.Enum(),
			RampUp:         proto.Int64(300000),
			IntervalMillis: proto.Int64(60000),
			Timeout:        proto.Int64(5000),
		},
		UnhealthyAt: proto.Int32(1),
	}
	deployment.PortMapping = []*protocol.PortMapping{
		&protocol.PortMapping{
			Scheme:      proto.String("HTTP"),
			PrivatePort: proto.Int32(8000),
			PublicPort:  proto.Int32(8000),
		},
	}
	return deployment, app
}

func TestSLAMonitor(t *testing.T) {

	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true
	logging.SetBackend(logBackend)
	logging.SetLevel(logging.ERROR, "")

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
		builder := builders.New(context.Config)
		scal := make(chan ChangeDeployCount)

		monitor := &simpleSLAMonitor{
			taskStore:    task_store.NewWithStore(ts),
			appStore:     app_store.NewWithStore(as),
			queue:        tq,
			interval:     0 * time.Nanosecond,
			needsScaling: scal,
		}
		monitor.Start()

		Reset(func() {
			tq.Stop()
			monitor.Stop()
			close(scal)
		})

		Convey("when nothing is queued or running", func() {

			_, component := AppWithSla(context, 1, 1, 1)

			Convey("it should need more instances", func() {
				So(monitor.NeedsMoreInstances(&component), ShouldBeTrue)
			})

			Convey("it can deploy more instances", func() {
				So(monitor.CanDeployMoreInstances(&component), ShouldBeTrue)
			})

			Convey("it should not need to change the deployment count", func() {
				So(monitor.changeDeployCount(), ShouldBeEmpty)
			})
		})

		Convey("when there are only apps queued, but nothing running yet", func() {
			Convey("for an app with 1 min instance and 1 max instance", func() {

				_, component := AppWithSla(context, 1, 1, 1)
				scheduled := test_utils.ScheduledComponent(&component)
				tq.Enqueue(&scheduled)
				monitor.appStore.Save(&component)

				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it can't deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it should not need to change the deployment count", func() {
					So(monitor.changeDeployCount(), ShouldBeEmpty)
				})
			})

			Convey("for an app with more max instances available", func() {
				_, component := AppWithSla(context, 1, 1, 3)
				scheduled := test_utils.ScheduledComponent(&component)
				tq.Enqueue(&scheduled)
				monitor.appStore.Save(&component)

				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it can deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeTrue)
				})

				Convey("it should not need to change the deployment count", func() {
					So(monitor.changeDeployCount(), ShouldBeEmpty)
				})
			})

			Convey("for an app with more min instances available", func() {
				_, component := AppWithSla(context, 1, 2, 3)
				scheduled := test_utils.ScheduledComponent(&component)
				tq.Enqueue(&scheduled)
				monitor.appStore.Save(&component)

				Convey("it should need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeTrue)
				})

				Convey("it can deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeTrue)
				})
				Convey("it should need to change the deployment count", func() {
					counts := monitor.changeDeployCount()
					So(counts, ShouldNotBeEmpty)
					So(counts[0].Count, ShouldEqual, 1)
				})
			})

			Convey("for an app with less max instances available", func() {

				_, component := AppWithSla(context, 1, 1, 1)
				scheduled := test_utils.ScheduledComponent(&component)
				tq.Enqueue(&scheduled)
				tq.Enqueue(&scheduled)
				monitor.appStore.Save(&component)
				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})
				Convey("it can't deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeFalse)
				})
				Convey("it should need to change the deployment count", func() {
					counts := monitor.changeDeployCount()
					So(counts, ShouldNotBeEmpty)
					So(counts[0].Count, ShouldEqual, -1)
				})
			})
		})

		Convey("with only running apps, nothing queued", func() {
			Convey("for an app with 1 min instance and 1 max instance", func() {

				deployment, component := AppWithSla(context, 1, 1, 1)
				monitor.taskStore.Save(&deployment)
				monitor.appStore.Save(&component)

				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it can't deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it should not need to change the deployment count", func() {
					So(monitor.changeDeployCount(), ShouldBeEmpty)
				})
			})

			Convey("for an app with more max instances available", func() {
				deployment, component := AppWithSla(context, 1, 1, 3)
				monitor.taskStore.Save(&deployment)
				monitor.appStore.Save(&component)

				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it can deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeTrue)
				})

				Convey("it should not need to change the deployment count", func() {
					So(monitor.changeDeployCount(), ShouldBeEmpty)
				})
			})

			Convey("for an app with more min instances available", func() {
				deployment, component := AppWithSla(context, 1, 2, 3)
				monitor.taskStore.Save(&deployment)
				monitor.appStore.Save(&component)

				Convey("it should need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeTrue)
				})

				Convey("it can deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeTrue)
				})
				Convey("it should need to change the deployment count", func() {
					counts := monitor.changeDeployCount()
					So(counts, ShouldNotBeEmpty)
					So(counts[0].Count, ShouldEqual, 1)
				})
			})

			Convey("for an app with less max instances available", func() {

				deployment, component := AppWithSla(context, 1, 1, 1)
				deployment.Status = protocol.AppStatus_STARTED.Enum()
				monitor.taskStore.Save(&deployment)

				offer := test_utils.CreateOffer("offer-4949", 1, 1024)
				scheduled := test_utils.ScheduledComponent(&component)
				task, _ := builder.BuildTaskInfo("task-94994", &offer, &scheduled)
				deployed := test_utils.DeployedApp(&component, &task)
				monitor.taskStore.Save(&deployed)
				monitor.appStore.Save(&component)

				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it can't deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it should need to change the deployment count", func() {
					counts := monitor.changeDeployCount()
					So(counts, ShouldNotBeEmpty)
					So(counts[0].Count, ShouldEqual, -1)
				})
			})
		})

		Convey("with both running apps, and things in the queue and receiving on a channel", func() {

			Convey("for an app with more min instances available", func() {
				deployment, component := AppWithSla(context, 1, 3, 4)
				scheduled := test_utils.ScheduledComponent(&component)
				tq.Enqueue(&scheduled)
				monitor.taskStore.Save(&deployment)
				monitor.appStore.Save(&component)

				Convey("it should need to change the deployment count", func() {
					go monitor.checkSLAConformance()
					cnt := <-monitor.needsScaling
					So(cnt.Count, ShouldEqual, 1)
				})
			})

			Convey("for an app with less max instances available", func() {
				deployment, component := AppWithSla(context, 1, 1, 1)
				deployment.Status = protocol.AppStatus_STARTED.Enum()
				monitor.taskStore.Save(&deployment)

				offer := test_utils.CreateOffer("offer-4949", 1, 1024)
				scheduled := test_utils.ScheduledComponent(&component)
				task, _ := builder.BuildTaskInfo("task-94994", &offer, &scheduled)
				deployed := test_utils.DeployedApp(&component, &task)
				monitor.taskStore.Save(&deployed)
				monitor.appStore.Save(&component)

				Convey("it should need to change the deployment count", func() {
					go monitor.checkSLAConformance()
					cnt := <-monitor.needsScaling
					So(cnt.Count, ShouldEqual, -1)
				})
			})
		})

		Convey("with both running apps, and things in the queue", func() {
			Convey("for an app with 1 min instance and 2 max instance", func() {

				deployment, component := AppWithSla(context, 1, 1, 2)
				scheduled := test_utils.ScheduledComponent(&component)
				tq.Enqueue(&scheduled)
				monitor.taskStore.Save(&deployment)
				monitor.appStore.Save(&component)

				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it can't deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it should not need to change the deployment count", func() {
					So(monitor.changeDeployCount(), ShouldBeEmpty)
				})
			})

			Convey("for an app with more max instances available", func() {
				deployment, component := AppWithSla(context, 1, 1, 3)
				scheduled := test_utils.ScheduledComponent(&component)
				tq.Enqueue(&scheduled)
				monitor.taskStore.Save(&deployment)
				monitor.appStore.Save(&component)

				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it can deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeTrue)
				})

				Convey("it should not need to change the deployment count", func() {
					So(monitor.changeDeployCount(), ShouldBeEmpty)
				})
			})

			Convey("for an app with more min instances available", func() {
				deployment, component := AppWithSla(context, 1, 3, 4)
				scheduled := test_utils.ScheduledComponent(&component)
				tq.Enqueue(&scheduled)
				monitor.taskStore.Save(&deployment)
				monitor.appStore.Save(&component)

				Convey("it should need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeTrue)
				})

				Convey("it can deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeTrue)
				})
				Convey("it should need to change the deployment count", func() {
					counts := monitor.changeDeployCount()
					So(counts, ShouldNotBeEmpty)
					So(counts[0].Count, ShouldEqual, 1)
				})
			})

			Convey("for an app with less max instances available", func() {

				deployment, component := AppWithSla(context, 1, 1, 1)
				deployment.Status = protocol.AppStatus_STARTED.Enum()
				monitor.taskStore.Save(&deployment)

				offer := test_utils.CreateOffer("offer-4949", 1, 1024)
				scheduled := test_utils.ScheduledComponent(&component)
				task, _ := builder.BuildTaskInfo("task-94994", &offer, &scheduled)
				deployed := test_utils.DeployedApp(&component, &task)
				monitor.taskStore.Save(&deployed)
				monitor.appStore.Save(&component)

				Convey("it should not need more instances", func() {
					So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it can't deploy more instances", func() {
					So(monitor.CanDeployMoreInstances(&component), ShouldBeFalse)
				})

				Convey("it should need to change the deployment count", func() {
					counts := monitor.changeDeployCount()
					So(counts, ShouldNotBeEmpty)
					So(counts[0].Count, ShouldEqual, -1)
				})
			})
		})

		Convey("when the application is inactive", func() {
			deployment, component := AppWithSla(context, 1, 1, 5)
			deployment.Status = protocol.AppStatus_STARTED.Enum()
			component.Active = proto.Bool(false)
			monitor.taskStore.Save(&deployment)
			monitor.appStore.Save(&component)

			Convey("it should not need more instances", func() {
				So(monitor.NeedsMoreInstances(&component), ShouldBeFalse)
			})

			Convey("it can't deploy more instances", func() {
				So(monitor.CanDeployMoreInstances(&component), ShouldBeFalse)
			})

			Convey("it should need to change the deployment count", func() {
				counts := monitor.changeDeployCount()
				So(counts, ShouldNotBeEmpty)
				So(counts[0].Count, ShouldEqual, -1)
			})
		})
	})
}
