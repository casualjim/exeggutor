package tasks

import (
	"fmt"
	"testing"

	"code.google.com/p/goprotobuf/proto"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	pth := fmt.Sprintf("../test-reports/junit_exeggutor_tasks_%d.xml", config.GinkgoConfig.ParallelNode)
	junitReporter := reporters.NewJUnitReporter(pth)
	RunSpecsWithDefaultAndCustomReporters(t, "Exeggutor Tasks Test Suite", []Reporter{junitReporter})
}

func testComponent(appName, compName string, cpus, mem float32) protocol.Application {
	distURL := "package://" + compName
	command := "./bin/" + compName
	version := "0.1.0"
	scheme := "http"
	var port int32 = 8000
	logs := "./logs"
	work := "./work"
	conf := "./conf"
	dist := protocol.Distribution_PACKAGE
	comp := protocol.ComponentType_SERVICE
	return protocol.Application{
		Name:          proto.String(compName),
		Cpus:          proto.Float32(cpus),
		Mem:           proto.Float32(mem),
		DistUrl:       proto.String(distURL),
		DiskSpace:     proto.Int64(0),
		Command:       proto.String(command),
		Version:       proto.String(version),
		AppName:       proto.String(appName),
		LogDir:        proto.String(logs),
		WorkDir:       proto.String(work),
		ConfDir:       proto.String(conf),
		Distribution:  &dist,
		ComponentType: &comp,
		Env:           nil, //[]*protocol.StringKeyValue{},
		Ports: []*protocol.StringIntKeyValue{
			&protocol.StringIntKeyValue{
				Key:   proto.String(scheme),
				Value: proto.Int32(port),
			},
		},
	}
}

func deployedApp(component *protocol.Application, task *mesos.TaskInfo) protocol.DeployedAppComponent {
	return protocol.DeployedAppComponent{
		AppName:   component.AppName,
		Component: component,
		TaskId:    task.TaskId,
		Status:    protocol.AppStatus_DEPLOYING.Enum(),
		Slave:     task.SlaveId,
	}
}

func scheduledComponent(component *protocol.Application) protocol.ScheduledApp {
	return protocol.ScheduledApp{
		Name:     component.Name,
		AppName:  component.AppName,
		App:      component,
		Position: proto.Int(0),
		Since:    proto.Int64(5),
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
			&mesos.Resource{
				Name: proto.String("ports"),
				Type: mesos.Value_RANGES.Enum(),
				Ranges: &mesos.Value_Ranges{
					Range: []*mesos.Value_Range{
						&mesos.Value_Range{
							Begin: proto.Uint64(uint64(32000)),
							End:   proto.Uint64(uint64(32999)),
						},
					},
				},
			},
		},
		Attributes:  []*mesos.Attribute{},
		ExecutorIds: []*mesos.ExecutorID{},
	}
}
