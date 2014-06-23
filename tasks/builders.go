// Package tasks provides ...
package tasks

import (
	"fmt"
	"strconv"
	"strings"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"
)

// BuildResources builds the []*mesos.Resource from a protocol.ApplicationComponent
func BuildResources(component *protocol.ApplicationComponent) []*mesos.Resource {
	return []*mesos.Resource{
		mesos.ScalarResource("cpus", float64(component.GetCpus())),
		mesos.ScalarResource("mem", float64(component.GetMem())),
	}
}

// BuildTaskEnvironment builds a mesos.Environment from the environment and ports
// provided by the application component
func BuildTaskEnvironment(envList []*protocol.StringKeyValue, ports []*protocol.StringIntKeyValue) *mesos.Environment {
	var env []*mesos.Environment_Variable
	for _, kv := range envList {
		env = append(env, &mesos.Environment_Variable{
			Name:  kv.Key,
			Value: kv.Value,
		})
	}
	for _, port := range ports {
		env = append(env, &mesos.Environment_Variable{
			Name:  proto.String(strings.ToUpper(port.GetKey()) + "_PORT"),
			Value: proto.String(strconv.Itoa(int(port.GetValue()))),
		})
	}
	return &mesos.Environment{Variables: env}
}

// BuildContainerInfo builds a mesos.ContainerInfo object from a protocol.ApplicationComponent
// It will only do this when the distribution is docker, otherwise it will return nil
func BuildContainerInfo(slaveID string, component *protocol.ApplicationComponent, ports PortProvider) *mesos.CommandInfo_ContainerInfo {
	if component.GetDistribution() != protocol.Distribution_DOCKER {
		return nil
	}
	var options []string
	for _, port := range component.Ports {
		p, _ := ports.Acquire(slaveID)
		options = append(options, fmt.Sprintf("--publish=%v:%v", p, port))
	}
	return &mesos.CommandInfo_ContainerInfo{
		Image:   proto.String("docker:///" + component.GetName() + ":" + component.GetVersion()),
		Options: options,
	}
}

// BuildMesosCommand builds a mesos.CommandInfo object from a protocol.ApplicationComponent
// This is what drives our deployment and how it works.
func BuildMesosCommand(slaveID string, component *protocol.ApplicationComponent, ports PortProvider) *mesos.CommandInfo {
	return &mesos.CommandInfo{
		Container:   BuildContainerInfo(slaveID, component, ports),
		Uris:        nil, // TODO: used to provide the docker image url for deimos?
		Environment: BuildTaskEnvironment(component.GetEnv(), component.GetPorts()),
		Value:       component.Command,
		User:        nil, // TODO: allow this to be configured?
		HealthCheck: nil, // TODO: allow this to be configured?
	}
}

// BuildTaskInfo builds a mesos.TaskInfo object from an offer and a scheduled component
func BuildTaskInfo(taskID string, offer *mesos.Offer, scheduled *protocol.ScheduledAppComponent, ports PortProvider) mesos.TaskInfo {
	component := scheduled.Component
	slaveID := offer.GetSlaveId().GetValue()

	return mesos.TaskInfo{
		Name:      scheduled.Name,
		TaskId:    &mesos.TaskID{Value: proto.String("exeggutor-task-" + taskID)},
		SlaveId:   offer.SlaveId,
		Command:   BuildMesosCommand(slaveID, component, ports),
		Resources: BuildResources(component),
		Executor:  nil, // TODO: Make use of an executor to increase visibility into execution
	}
}
