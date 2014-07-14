package main

import (
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"time"

	"code.google.com/p/goprotobuf/proto"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/health"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"
)

func makeDeployedApp(port int, name, component, id string, delay, interval, timeout int64) (deployment *protocol.Deployment, app *protocol.Application) {
	app = &protocol.Application{
		Id:            proto.String(strings.Join([]string{name, component, "0.0.1"}, "-")),
		Name:          proto.String(component),
		Cpus:          proto.Float32(1),
		Mem:           proto.Float32(64),
		DistUrl:       proto.String("docker://blah"),
		DiskSpace:     proto.Int64(0),
		Command:       proto.String("bin/blah"),
		Version:       proto.String("0.0.1"),
		AppName:       proto.String(name),
		LogDir:        proto.String("./logs"),
		WorkDir:       proto.String("./work"),
		ConfDir:       proto.String("./conf"),
		Distribution:  protocol.Distribution_DOCKER.Enum(),
		ComponentType: protocol.ComponentType_SERVICE.Enum(),
		Env:           nil, //[]*protocol.StringKeyValue{},
		Ports: []*protocol.StringIntKeyValue{
			&protocol.StringIntKeyValue{
				Key:   proto.String("HTTP"),
				Value: proto.Int(port),
			},
		},
		Sla: &protocol.ApplicationSLA{
			MinInstances: proto.Int32(1),
			MaxInstances: proto.Int32(1),
			UnhealthyAt:  proto.Int32(1),
			HealthCheck: &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(delay),
				IntervalMillis: proto.Int64(interval),
				Timeout:        proto.Int64(timeout),
			},
		},
	}
	deployment = &protocol.Deployment{

		AppId:    proto.String(app.ID()),
		HostName: proto.String("127.0.0.1"),
		TaskId: &mesos.TaskID{
			Value: proto.String(id),
		},
		PortMapping: []*protocol.PortMapping{
			&protocol.PortMapping{
				Scheme:      proto.String("HTTP"),
				PrivatePort: proto.Int(port),
				PublicPort:  proto.Int(port),
			},
		},
	}
	return
}

var log = logging.MustGetLogger("exeggutor.playground")

// func MaxParallelism() int {
// 	maxProcs := runtime.GOMAXPROCS(0)
// 	numCPU := runtime.NumCPU()
// 	if maxProcs < numCPU {
// 		return maxProcs
// 	}
// 	return numCPU
// }
func main() {
	// runtime.GOMAXPROCS(MaxParallelism())
	logBackend := logging.NewLogBackend(os.Stderr, "", stdlog.LstdFlags|stdlog.Lshortfile)
	logBackend.Color = true
	logging.SetBackend(logBackend)
	logging.SetLevel(logging.DEBUG, "")
	context := &exeggutor.AppContext{
		Config: &exeggutor.Config{
			FrameworkInfo: &exeggutor.FrameworkConfig{
				HealthCheckConcurrency: 50,
			},
		},
	}
	handler := func(port int) http.Handler {
		counter := 0
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			counter++
			if counter%5 == 0 {
				rw.WriteHeader(http.StatusInternalServerError)
			} else {
				rw.WriteHeader(http.StatusOK)
			}
		})
	}

	port1 := 39393
	port2 := 8472
	port3 := 4482
	port4 := 3922
	go func() {
		log.Notice("Starting app 1 on %d", port1)
		http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port1), handler(port1))
	}()
	go func() {
		log.Notice("Starting app 2 on %d", port2)
		http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port2), handler(port2))
	}()
	go func() {
		log.Notice("Starting app 3 on %d", port3)
		http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port3), handler(port3))
	}()
	go func() {
		log.Notice("Starting app 4 on %d", port4)
		http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port4), handler(port4))
	}()

	checker := health.New(context)
	checker.Start()
	go func() {
		for i := 0; i < 10; i++ {
			checker.Register(makeDeployedApp(port1, "application-1", fmt.Sprintf("comp-%d-app-1", i), fmt.Sprintf("task-%d-app-1", i), 5000, 1000, 1000))
			checker.Register(makeDeployedApp(port2, "application-2", fmt.Sprintf("comp-%d-app-1", i), fmt.Sprintf("task-%d-app-2", i), 3000, 1000, 1000))
			checker.Register(makeDeployedApp(port3, "application-3", fmt.Sprintf("comp-%d-app-1", i), fmt.Sprintf("task-%d-app-3", i), 10000, 1000, 1000))
			checker.Register(makeDeployedApp(port4, "application-4", fmt.Sprintf("comp-%d-app-1", i), fmt.Sprintf("task-%d-app-4", i), 20000, 1000, 1000))
		}
		log.Info("Scheduled 40 health checks to run")
	}()

	go func() {
		for result := range checker.Failures() {
			log.Info("********   Received result %+v", result)
		}
	}()

	<-time.After(1 * time.Minute)
	log.Info("1 minute elapsed, stopping")
	checker.Stop()
	os.Exit(0)
}
