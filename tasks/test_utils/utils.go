package test_utils

import (
	"strconv"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/exeggutor/tasks/builders"
	"github.com/reverb/go-mesos/mesos"
)

func CreateFilterData(ts store.KVStore, b *builders.MesosMessageBuilder) []protocol.DeployedAppComponent {
	app1 := BuildStoreTestData2(1, 1, 1, b)
	app2 := BuildStoreTestData2(1, 2, 3, b)
	app3 := BuildStoreTestData2(1, 3, 2, b)
	app4 := BuildStoreTestData2(2, 1, 2, b)
	app5 := BuildStoreTestData2(3, 1, 3, b)
	app6 := BuildStoreTestData2(2, 1, 3, b)
	SaveStoreTestData(ts, &app1)
	SaveStoreTestData(ts, &app2)
	SaveStoreTestData(ts, &app3)
	SaveStoreTestData(ts, &app4)
	SaveStoreTestData(ts, &app5)
	SaveStoreTestData(ts, &app6)
	return []protocol.DeployedAppComponent{app1, app2, app3, app4, app5, app6}
}

func SetupCallbackTestData(ts store.KVStore, b *builders.MesosMessageBuilder) (*mesos.TaskID, protocol.DeployedAppComponent) {
	offer := CreateOffer("offer id", 1.0, 64.0)
	component := TestComponent("app name", "component name", 1.0, 64.0)
	cr := &component
	scheduled := ScheduledComponent(cr)
	task := b.BuildTaskInfo("task id", &offer, &scheduled)
	tr := &task
	id := task.GetTaskId()
	deployed := DeployedApp(cr, tr)
	bytes, _ := proto.Marshal(&deployed)
	ts.Set(id.GetValue(), bytes)

	return id, deployed
}

func CreateStoreTestData(backing store.KVStore, b *builders.MesosMessageBuilder) (*mesos.TaskID, protocol.DeployedAppComponent) {

	deployed := BuildStoreTestData(1, b)
	SaveStoreTestData(backing, &deployed)
	return deployed.TaskId, deployed
}

func CreateMulti(backing store.KVStore, b *builders.MesosMessageBuilder) []protocol.DeployedAppComponent {
	app1 := BuildStoreTestData(1, b)
	app2 := BuildStoreTestData(2, b)
	app3 := BuildStoreTestData(3, b)
	SaveStoreTestData(backing, &app1)
	SaveStoreTestData(backing, &app2)
	SaveStoreTestData(backing, &app3)
	return []protocol.DeployedAppComponent{app1, app2, app3}
}

func BuildStoreTestData(index int, b *builders.MesosMessageBuilder) protocol.DeployedAppComponent {
	component := TestComponent("app-store-"+strconv.Itoa(index), "app-"+strconv.Itoa(index), 1, 64)
	scheduled := ScheduledComponent(&component)
	offer := CreateOffer("slave-"+strconv.Itoa(index), 8, 1024)
	task := b.BuildTaskInfo("task-app-id-"+strconv.Itoa(index), &offer, &scheduled)
	return DeployedApp(&component, &task)
}
func BuildStoreTestData2(app, componentID, taskID int, b *builders.MesosMessageBuilder) protocol.DeployedAppComponent {
	component := TestComponent("app-store-"+strconv.Itoa(app), "app-"+strconv.Itoa(componentID), 1, 64)
	scheduled := ScheduledComponent(&component)
	offer := CreateOffer("slave-"+strconv.Itoa(taskID), 8, 1024)
	task := b.BuildTaskInfo("task-app-"+strconv.Itoa(app)+"-"+strconv.Itoa(componentID)+"-id-"+strconv.Itoa(taskID), &offer, &scheduled)
	return DeployedApp(&component, &task)
}

func SaveStoreTestData(backing store.KVStore, deployed *protocol.DeployedAppComponent) {

	bytes, _ := proto.Marshal(deployed)
	backing.Set(deployed.TaskId.GetValue(), bytes)

}

func TestComponent(appName, compName string, cpus, mem float32) protocol.Application {
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

func DeployedApp(component *protocol.Application, task *mesos.TaskInfo) protocol.DeployedAppComponent {
	return protocol.DeployedAppComponent{
		AppName:   component.AppName,
		Component: component,
		TaskId:    task.TaskId,
		Status:    protocol.AppStatus_DEPLOYING.Enum(),
		Slave:     task.SlaveId,
	}
}

func ScheduledComponent(component *protocol.Application) protocol.ScheduledApp {
	return protocol.ScheduledApp{
		Name:     component.Name,
		AppName:  component.AppName,
		App:      component,
		Position: proto.Int(0),
		Since:    proto.Int64(5),
	}
}

func CreateOffer(id string, cpus, mem float64) mesos.Offer {
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
