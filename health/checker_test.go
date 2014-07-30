package health

import (
	stdlog "log"
	"os"
	"testing"
	"time"

	"code.google.com/p/goprotobuf/proto"

	// . "github.com/reverb/go-utils/convey/matchers"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/tasks/builders"
	"github.com/reverb/exeggutor/test_utils"
	"github.com/reverb/go-utils/flake"
	. "github.com/smartystreets/goconvey/convey"
)

func AppWithHealthCheck(context *exeggutor.AppContext, index int, delay, interval, timeout int64) (protocol.Deployment, protocol.Application) {
	deployment, app := test_utils.BuildStoreTestData(index, builders.New(context.Config))
	app.Sla = &protocol.ApplicationSLA{
		MinInstances: proto.Int32(1),
		MaxInstances: proto.Int32(1),
		HealthCheck: &protocol.HealthCheck{
			Mode:           protocol.HealthCheckMode_HTTP.Enum(),
			RampUp:         proto.Int64(delay),
			IntervalMillis: proto.Int64(interval),
			Timeout:        proto.Int64(timeout),
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
func TestHealthChecker(t *testing.T) {

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

	Convey("A HealthChecker", t, func() {

		builder := builders.New(context.Config)
		builder.PortPicker = &test_utils.ConstantPortPicker{Port: 8000}

		checker := New(context)
		checker.Start()

		Reset(func() {
			checker.Stop()
		})

		Convey("when registering invalid values", func() {

			Convey("return nil for a nil SLA", func() {
				deployment, app := test_utils.BuildStoreTestData(1, builders.New(context.Config))
				err := checker.Register(&deployment, &app)
				So(err, ShouldBeNil)
			})

			Convey("return nil for a nil health check", func() {
				deployment, app := test_utils.BuildStoreTestData(1, builders.New(context.Config))
				app.Sla = &protocol.ApplicationSLA{}
				err := checker.Register(&deployment, &app)
				So(err, ShouldBeNil)
			})

			Convey("return nil for a nil port mapping", func() {
				deployment, app := test_utils.BuildStoreTestData(1, builders.New(context.Config))
				app.Sla = &protocol.ApplicationSLA{HealthCheck: &protocol.HealthCheck{}}
				err := checker.Register(&deployment, &app)
				So(err, ShouldBeNil)
			})
		})

		Convey("when registering valid values", func() {

			deployment, app := AppWithHealthCheck(context, 1, 300000, 60000, 5000)

			Convey("and the value is new", func() {
				err := checker.Register(&deployment, &app)
				So(err, ShouldBeNil)
				val, ok := checker.register[deployment.GetTaskId().GetValue()]

				Convey("should store the value in the registry", func() {
					So(ok, ShouldBeTrue)
					So(val.ExpiresAt, ShouldHappenAfter, time.Now())
					So(len(checker.register), ShouldEqual, 1)
					So(checker.queue.Len(), ShouldEqual, 1)
				})

				Convey("should enqueue the value in the priority queue", func() {
					So(checker.queue.queue, ShouldContain, val)
				})
			})

			Convey("and the value exits", func() {
				checker.Register(&deployment, &app)
				val, _ := checker.register[deployment.GetTaskId().GetValue()]
				deployment3, app3 := AppWithHealthCheck(context, 2, 150000, 5000, 500)
				checker.Register(&deployment3, &app3)

				deployment4, app4 := AppWithHealthCheck(context, 3, 450000, 5000, 500)
				checker.Register(&deployment4, &app4)
				deployment2, app2 := AppWithHealthCheck(context, 1, 300000, 5000, 500)

				Convey("it should not lose its place in the queue", func() {
					var previousIndex int
					for i, ac := range checker.queue.queue {
						if ac.HealthCheck.GetID() == val.HealthCheck.GetID() {
							previousIndex = i
							break
						}
					}

					err := checker.Register(&deployment2, &app2)
					So(err, ShouldBeNil)
					var currentIndex int
					for i, ac := range checker.queue.queue {
						if ac.HealthCheck.GetID() == val.HealthCheck.GetID() {
							currentIndex = i
							break
						}
					}
					So(currentIndex, ShouldEqual, previousIndex)
				})

				Convey("it should update the values", func() {
					err := checker.Register(&deployment2, &app2)
					So(err, ShouldBeNil)
					var check *activeHealthCheck
					for _, ac := range checker.queue.queue {
						if ac.HealthCheck.GetID() == val.HealthCheck.GetID() {
							check = ac
							break
						}
					}
					So(check, ShouldNotBeNil)
				})
			})
		})

		Convey("when unregistering", func() {
			d, app := AppWithHealthCheck(context, 10, 300000, 60000, 5000)
			d2, app2 := AppWithHealthCheck(context, 20, 150000, 60000, 5000)
			d3, app3 := AppWithHealthCheck(context, 30, 450000, 60000, 5000)
			d4, app4 := AppWithHealthCheck(context, 40, 100000, 60000, 5000)

			checker.Register(&d, &app)
			checker.Register(&d2, &app2)
			checker.Register(&d3, &app3)
			checker.Register(&d4, &app4)

			So(len(checker.register), ShouldEqual, 4)
			So(checker.queue.Len(), ShouldEqual, 4)

			_, ok := checker.register[d.TaskId.GetValue()]
			So(ok, ShouldBeTrue)
			err := checker.Unregister(d.TaskId)
			So(err, ShouldBeNil)

			Convey("it should remove the check from the registry", func() {

				_, ok := checker.register[d.TaskId.GetValue()]
				So(ok, ShouldBeFalse)
				So(len(checker.register), ShouldEqual, 3)
			})

			Convey("it should remove the check from the queue", func() {
				found := checker.queue.Contains(d.TaskId.GetValue())
				So(found, ShouldBeFalse)
				So(checker.queue.Len(), ShouldEqual, 3)
			})
		})
	})

}
