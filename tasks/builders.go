// Package tasks provides ...
package tasks

import (
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

// BuildMesosCommand builds a mesos.CommandInfo object from a protocol.ApplicationComponent
// This is what drives our deployment and how it works.
func BuildMesosCommand(component *protocol.ApplicationComponent) *mesos.CommandInfo {
	return &mesos.CommandInfo{
		Container:   nil,                        // TODO: use this to configure deimos
		Uris:        []*mesos.CommandInfo_URI{}, // TODO: used to provide the docker image url for deimos?
		Environment: BuildTaskEnvironment(component.GetEnv(), component.GetPorts()),
		Value:       component.Command,
		User:        nil, // TODO: allow this to be configured?
		HealthCheck: nil, // TODO: allow this to be configured?
	}
}
