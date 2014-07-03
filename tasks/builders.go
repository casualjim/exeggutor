// Package tasks provides ...
package tasks

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"
)

type portRange struct {
	Begin uint64
	End   uint64
}

// BuildResources builds the []*mesos.Resource from a protocol.ApplicationComponent
func BuildResources(component *protocol.Application, ports []portRange) []*mesos.Resource {
	var pres []*mesos.Value_Range

	for _, port := range ports {
		pres = append(pres, &mesos.Value_Range{
			Begin: proto.Uint64(port.Begin),
			End:   proto.Uint64(port.End),
		})
	}

	return []*mesos.Resource{
		mesos.ScalarResource("cpus", float64(component.GetCpus())),
		mesos.ScalarResource("mem", float64(component.GetMem())),
		&mesos.Resource{
			Name:   proto.String("ports"),
			Type:   mesos.Value_RANGES.Enum(),
			Ranges: &mesos.Value_Ranges{Range: pres},
		},
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
func BuildContainerInfo(slaveID string, component *protocol.Application) *mesos.CommandInfo_ContainerInfo {
	// av := ports
	if component.GetDistribution() != protocol.Distribution_DOCKER {
		return nil
	}
	var options []string
	// for _, port := range component.Ports {
	// 	p, pp := av[0], av[1:]
	// 	av = pp
	// 	options = append(options, "-p", fmt.Sprintf("%v:%v", p, port.GetValue()))
	// }
	return &mesos.CommandInfo_ContainerInfo{
		Image:   proto.String("docker:///casualjim/" + component.GetName() + ":" + component.GetVersion()),
		Options: options,
	}
}

// BuildMesosCommand builds a mesos.CommandInfo object from a protocol.ApplicationComponent
// This is what drives our deployment and how it works.
func BuildMesosCommand(slaveID string, component *protocol.Application) *mesos.CommandInfo {

	return &mesos.CommandInfo{
		Container:   BuildContainerInfo(slaveID, component),
		Uris:        nil, // TODO: used to provide the docker image url for deimos?
		Environment: BuildTaskEnvironment(component.GetEnv(), component.GetPorts()),
		Value:       component.Command,
		User:        nil, // TODO: allow this to be configured?
		HealthCheck: nil, // TODO: allow this to be configured?
	}
}

func makePortRange(min, max, count int) portRange {
	// we will try to randomize port assignment a little by
	// looking at the range, calculating how near to the end we have to stop
	// at the very latest. Then we take the begin port and add a random value
	// that doesn't exceed the maximum we calculated earlier
	// We use that random value as our begin port and take as many sequential ports
	// as we need at that point.

	// inclusive, and adds an extra one to compensate for rand later on so +2 instead of +1
	maxValidBeginPort := max - min - count + 2
	rand.Seed(time.Now().Unix())
	beginPort := rand.Intn(maxValidBeginPort) + min

	return portRange{
		Begin: uint64(beginPort),
		End:   uint64(int(beginPort) + count - 1), // exclusive
	}
}

func getPorts(offer *mesos.Offer, count int) []portRange {
	untaken := count
	var takenRanges []portRange
	// TODO: deal with the case where the ranges are too fragmented
	// which is probably never because we have at least 1000 ports
	// to begin with, and our hosts are relatively small

	// get all the resources
	for _, resource := range offer.GetResources() {
		// we just want the ports here
		if resource.GetName() == "ports" {
			// unwrap the ranges
			for _, r := range resource.GetRanges().GetRange() {
				// when we have stuff left to take, take it, this is inclusive
				numAvail := int(r.GetEnd() - r.GetBegin() + 1)
				if untaken < numAvail { // this fits in this range
					prange := makePortRange(int(r.GetBegin()), int(r.GetEnd()), untaken)
					takenRanges = append(takenRanges, prange)
					untaken = 0
				}
				if untaken == 0 { // stop as soon as we're at 0
					break
				}
			}
		}
		if untaken == 0 { // we're done here
			break
		}
	}
	return takenRanges
}

// BuildTaskInfo builds a mesos.TaskInfo object from an offer and a scheduled component
func BuildTaskInfo(taskID string, offer *mesos.Offer, scheduled *protocol.ScheduledApp) mesos.TaskInfo {
	component := scheduled.App
	slaveID := offer.GetSlaveId().GetValue()

	return mesos.TaskInfo{
		Name:      proto.String(scheduled.GetAppName() + "-" + scheduled.GetName()),
		TaskId:    &mesos.TaskID{Value: proto.String("exeggutor-task-" + taskID)},
		SlaveId:   offer.SlaveId,
		Command:   BuildMesosCommand(slaveID, component),
		Resources: BuildResources(component, getPorts(offer, len(component.GetPorts()))),
		Executor:  nil, // TODO: Make use of an executor to increase visibility into execution
	}
}
