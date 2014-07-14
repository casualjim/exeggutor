package test_utils

import (
	"strconv"
	"strings"
	"time"

	"code.google.com/p/goprotobuf/proto"

	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/exeggutor/tasks/builders"
	"github.com/reverb/go-mesos/mesos"
)

type ConstantPortPicker struct {
	Port int
}

func (p *ConstantPortPicker) GetPorts(offer *mesos.Offer, count int) ([]builders.PortRange, []int32) {
	return builders.PortRangeFor(p.Port)
}

func CreateFilterData(ts store.KVStore, as store.KVStore, b *builders.MesosMessageBuilder) ([]protocol.Deployment, []protocol.Application) {
	d1, app1 := BuildStoreTestData2(1, 1, 1, b)
	d2, app2 := BuildStoreTestData2(1, 2, 3, b)
	d3, app3 := BuildStoreTestData2(1, 3, 2, b)
	d4, app4 := BuildStoreTestData2(2, 1, 2, b)
	d5, app5 := BuildStoreTestData2(3, 1, 3, b)
	d6, app6 := BuildStoreTestData2(2, 1, 3, b)
	SaveStoreTestData(ts, as, &d1, &app1)
	SaveStoreTestData(ts, as, &d2, &app2)
	SaveStoreTestData(ts, as, &d3, &app3)
	SaveStoreTestData(ts, as, &d4, &app4)
	SaveStoreTestData(ts, as, &d5, &app5)
	SaveStoreTestData(ts, as, &d6, &app6)
	return []protocol.Deployment{d1, d2, d3, d4, d5, d6}, []protocol.Application{app1, app2, app3, app4, app5, app6}
}

func SetupCallbackTestData(ts store.KVStore, as store.KVStore, b *builders.MesosMessageBuilder) (*mesos.TaskID, protocol.Deployment, protocol.Application) {
	offer := CreateOffer("offer id", 1.0, 64.0)
	component := TestComponent("app name", "component name", 1.0, 64.0)
	cr := &component
	scheduled := ScheduledComponent(cr)
	task, _ := b.BuildTaskInfo("task id", &offer, &scheduled)
	tr := &task
	id := task.GetTaskId()
	deployed := DeployedApp(cr, tr)
	bytes, _ := proto.Marshal(&deployed)
	ts.Set(id.GetValue(), bytes)
	ab, _ := proto.Marshal(&component)
	as.Set(component.GetId(), ab)

	return id, deployed, component
}

func CreateStoreTestData(backing store.KVStore, b *builders.MesosMessageBuilder) (*mesos.TaskID, protocol.Deployment) {
	deployed, _ := BuildStoreTestData(1, b)
	bytes, _ := proto.Marshal(&deployed)
	backing.Set(deployed.TaskId.GetValue(), bytes)
	return deployed.TaskId, deployed
}

func CreateAppStoreTestData(backing store.KVStore, b *builders.MesosMessageBuilder) protocol.Application {
	_, app := BuildStoreTestData(1, b)
	bytes, _ := proto.Marshal(&app)
	backing.Set(app.GetId(), bytes)
	return app
}

func CreateMulti(backing store.KVStore, b *builders.MesosMessageBuilder) []protocol.Deployment {
	d1, _ := BuildStoreTestData(1, b)
	d2, _ := BuildStoreTestData(2, b)
	d3, _ := BuildStoreTestData(3, b)
	b1, _ := proto.Marshal(&d1)
	b2, _ := proto.Marshal(&d2)
	b3, _ := proto.Marshal(&d3)
	backing.Set(d1.GetTaskId().GetValue(), b1)
	backing.Set(d2.GetTaskId().GetValue(), b2)
	backing.Set(d3.GetTaskId().GetValue(), b3)
	return []protocol.Deployment{d1, d2, d3}
}

func CreateAppStoreMulti(backing store.KVStore, b *builders.MesosMessageBuilder) []protocol.Application {
	_, app1 := BuildStoreTestData(1, b)
	_, app2 := BuildStoreTestData(2, b)
	_, app3 := BuildStoreTestData(3, b)
	b1, _ := proto.Marshal(&app1)
	b2, _ := proto.Marshal(&app2)
	b3, _ := proto.Marshal(&app3)
	backing.Set(app1.GetId(), b1)
	backing.Set(app2.GetId(), b2)
	backing.Set(app3.GetId(), b3)
	return []protocol.Application{app1, app2, app3}
}

func BuildStoreTestData(index int, b *builders.MesosMessageBuilder) (protocol.Deployment, protocol.Application) {
	component := TestComponent("app-store-"+strconv.Itoa(index), "app-"+strconv.Itoa(index), 1, 64)
	scheduled := ScheduledComponent(&component)
	offer := CreateOffer("slave-"+strconv.Itoa(index), 8, 1024)
	task, _ := b.BuildTaskInfo("task-app-id-"+strconv.Itoa(index), &offer, &scheduled)
	return DeployedApp(&component, &task), component
}
func BuildStoreTestData2(app, componentID, taskID int, b *builders.MesosMessageBuilder) (protocol.Deployment, protocol.Application) {
	component := TestComponent("app-store-"+strconv.Itoa(app), "app-"+strconv.Itoa(componentID), 1, 64)
	scheduled := ScheduledComponent(&component)
	offer := CreateOffer("slave-"+strconv.Itoa(taskID), 8, 1024)
	task, _ := b.BuildTaskInfo("task-app-"+strconv.Itoa(app)+"-"+strconv.Itoa(componentID)+"-id-"+strconv.Itoa(taskID), &offer, &scheduled)
	return DeployedApp(&component, &task), component
}

func SaveStoreTestData(backing store.KVStore, as store.KVStore, deployed *protocol.Deployment, app *protocol.Application) {

	bytes, _ := proto.Marshal(deployed)
	backing.Set(deployed.TaskId.GetValue(), bytes)
	appb, _ := proto.Marshal(app)
	as.Set(app.GetId(), appb)
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
		Id:            proto.String(strings.Join([]string{appName, compName, version}, "-")),
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
		Active:        proto.Bool(true),
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

func DeployedApp(component *protocol.Application, task *mesos.TaskInfo) protocol.Deployment {
	return protocol.Deployment{
		AppId:       component.Id,
		TaskId:      task.TaskId,
		Status:      protocol.AppStatus_DEPLOYING.Enum(),
		Slave:       task.SlaveId,
		HostName:    proto.String("exeggutor-slave-instance-1"),
		PortMapping: nil,
		DeployedAt:  proto.Int64(time.Now().UnixNano() / 1000000),
	}
}

func ScheduledComponent(component *protocol.Application) protocol.ScheduledApp {
	return protocol.ScheduledApp{
		AppId:    component.Id,
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
