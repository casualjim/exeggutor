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
	"github.com/reverb/exeggutor/tasks/test_utils"
	"github.com/reverb/go-utils/flake"
	. "github.com/smartystreets/goconvey/convey"
)

func appWithHealthCheck(context *exeggutor.AppContext, index int, delay, interval, timeout int64) protocol.DeployedAppComponent {
	app := test_utils.BuildStoreTestData(index, builders.New(context.Config))
	app.Component.Sla = &protocol.ApplicationSLA{
		MinInstances: proto.Int32(1),
		MaxInstances: proto.Int32(1),
		HealthCheck: &protocol.HealthCheck{
			Mode:           protocol.HealthCheckMode_HTTP.Enum(),
			RampUp:         proto.Int64(delay),    // 5 min
			IntervalMillis: proto.Int64(interval), // 1 min
			Timeout:        proto.Int64(timeout),  // 5 sec
		},
		UnhealthyAt: proto.Int32(1),
	}
	app.PortMapping = []*protocol.PortMapping{
		&protocol.PortMapping{
			Scheme:      proto.String("HTTP"),
			PrivatePort: proto.Int32(8000),
			PublicPort:  proto.Int32(8000),
		},
	}
	return app
}

func TestHealthChecker(t *testing.T) {

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

	Convey("A HealthChecker", t, func() {

		builder := builders.New(context.Config)
		builder.PortPicker = &test_utils.ConstantPortPicker{Port: 8000}

		checker := New(context)
		checker.Start()

		Reset(func() {
			checker.Stop()
		})

		Convey("when registering invalid values", func() {

			Convey("return an error for nil app component", func() {
				app := test_utils.BuildStoreTestData(1, builder)
				app.Component = nil
				err := checker.Register(&app)
				So(err, ShouldNotBeNil)
			})

			Convey("return nil for a nil SLA", func() {
				app := test_utils.BuildStoreTestData(1, builders.New(context.Config))
				err := checker.Register(&app)
				So(err, ShouldBeNil)
			})

			Convey("return nil for a nil health check", func() {
				app := test_utils.BuildStoreTestData(1, builders.New(context.Config))
				app.Component.Sla = &protocol.ApplicationSLA{}
				err := checker.Register(&app)
				So(err, ShouldBeNil)
			})

			Convey("return nil for a nil port mapping", func() {
				app := test_utils.BuildStoreTestData(1, builders.New(context.Config))
				app.Component.Sla = &protocol.ApplicationSLA{HealthCheck: &protocol.HealthCheck{}}
				err := checker.Register(&app)
				So(err, ShouldBeNil)
			})
		})

		Convey("when registering valid values", func() {

			app := appWithHealthCheck(context, 1, 300000, 60000, 5000)

			Convey("and the value is new", func() {
				err := checker.Register(&app)
				So(err, ShouldBeNil)
				val, ok := checker.register[app.GetTaskId().GetValue()]

				Convey("should store the value in the registry", func() {
					So(ok, ShouldBeTrue)
					So(val.ExpiresAt, ShouldHappenAfter, time.Now())
					So(len(checker.register), ShouldEqual, 1)
					So(len(checker.queue), ShouldEqual, 1)
				})

				Convey("should enqueue the value in the priority queue", func() {
					So(checker.queue, ShouldContain, val)
				})
			})

			Convey("and the value exits", func() {
				checker.Register(&app)
				val, _ := checker.register[app.GetTaskId().GetValue()]
				app3 := appWithHealthCheck(context, 2, 150000, 5000, 500)
				checker.Register(&app3)

				app4 := appWithHealthCheck(context, 3, 450000, 5000, 500)
				checker.Register(&app4)
				app2 := appWithHealthCheck(context, 1, 300000, 5000, 500)

				Convey("it should not lose its place in the queue", func() {
					var previousIndex int
					for i, ac := range checker.queue {
						if ac.HealthCheck.GetID() == val.HealthCheck.GetID() {
							previousIndex = i
							break
						}
					}

					err := checker.Register(&app2)
					So(err, ShouldBeNil)
					var currentIndex int
					for i, ac := range checker.queue {
						if ac.HealthCheck.GetID() == val.HealthCheck.GetID() {
							currentIndex = i
							break
						}
					}
					So(currentIndex, ShouldEqual, previousIndex)
				})

				Convey("it should update the values", func() {
					err := checker.Register(&app2)
					So(err, ShouldBeNil)
					var check *activeHealthCheck
					for _, ac := range checker.queue {
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
			app := appWithHealthCheck(context, 1, 300000, 60000, 5000)

			app2 := appWithHealthCheck(context, 2, 150000, 60000, 5000)
			app3 := appWithHealthCheck(context, 3, 450000, 60000, 5000)
			app4 := appWithHealthCheck(context, 4, 100000, 60000, 5000)

			checker.Register(&app)
			checker.Register(&app2)
			checker.Register(&app3)
			checker.Register(&app4)

			So(len(checker.register), ShouldEqual, 4)
			So(len(checker.queue), ShouldEqual, 4)

			val, ok := checker.register[app.TaskId.GetValue()]
			So(ok, ShouldBeTrue)
			err := checker.Unregister(app.TaskId)
			So(err, ShouldBeNil)

			Convey("it should remove the check from the registry", func() {

				_, ok := checker.register[app.TaskId.GetValue()]
				So(ok, ShouldBeFalse)
				So(len(checker.register), ShouldEqual, 3)
			})

			Convey("it should remove the check from the queue", func() {
				found := false
				for _, ac := range checker.queue {
					if ac.HealthCheck.GetID() == val.HealthCheck.GetID() {
						found = true
					}
				}
				So(found, ShouldBeFalse)
				So(len(checker.queue), ShouldEqual, 3)
			})
		})
	})

}
