package builders

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.tasks.builders")

type PortRange struct {
	Begin uint64
	End   uint64
}

var EmptyPortRange = []PortRange{}

func PortRangeFor(port int) ([]PortRange, []int32) {
	return []PortRange{PortRange{Begin: uint64(port), End: uint64(port)}}, []int32{int32(port)}
}

// PortPicker provides an object that picks ports from a port range in a mesos offer
type PortPicker interface {
	GetPorts(*mesos.Offer, int) ([]PortRange, []int32)
}

// The default implementation picks a random start port for a range
// slicing a port range into random pieces
type RandomPortPicker struct {
}

func (p *RandomPortPicker) makePortRange(min, max, count int) PortRange {
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

	return PortRange{
		Begin: uint64(beginPort),
		End:   uint64(int(beginPort) + count - 1), // exclusive
	}
}

// GetPorts provides a new valid port range from the available ports in the offer
func (p *RandomPortPicker) GetPorts(offer *mesos.Offer, count int) (takenRanges []PortRange, reservedPorts []int32) {
	untaken := count
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
					prange := p.makePortRange(int(r.GetBegin()), int(r.GetEnd()), untaken)
					takenRanges = append(takenRanges, prange)
					untaken = 0
					for i := prange.Begin; i <= prange.End; i++ {
						reservedPorts = append(reservedPorts, int32(i))
					}
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
	return takenRanges, reservedPorts
}

// MesosMessageBuilder converts internal messages into a mesos messages
type MesosMessageBuilder struct {
	config     *exeggutor.Config
	PortPicker PortPicker
}

// New creates a new instance of the message builder with the specified config
func New(config *exeggutor.Config) *MesosMessageBuilder {
	return &MesosMessageBuilder{config: config, PortPicker: &RandomPortPicker{}}
}

// BuildResources builds the []*mesos.Resource from a protocol.ApplicationComponent
func (b *MesosMessageBuilder) BuildResources(component *protocol.Application, ports []PortRange) []*mesos.Resource {
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
func (b *MesosMessageBuilder) BuildTaskEnvironment(envList []*protocol.StringKeyValue, ports []*protocol.StringIntKeyValue, reservedPorts []int32) *mesos.Environment {
	var env []*mesos.Environment_Variable
	for _, kv := range envList {
		env = append(env, &mesos.Environment_Variable{
			Name:  kv.Key,
			Value: kv.Value,
		})
	}
	for i, port := range ports {
		env = append(
			env,
			&mesos.Environment_Variable{
				Name:  proto.String(strings.ToUpper(port.GetKey()) + "_PORT"),
				Value: proto.String(strconv.Itoa(int(port.GetValue()))),
			},
			&mesos.Environment_Variable{
				Name:  proto.String(strings.ToUpper(port.GetKey()) + "_PUBLIC_PORT"),
				Value: proto.String(strconv.Itoa(int(reservedPorts[i]))),
			},
		)
	}
	return &mesos.Environment{Variables: env}
}

// BuildContainerInfo builds a mesos.ContainerInfo object from a protocol.ApplicationComponent
// It will only do this when the distribution is docker, otherwise it will return nil
func (b *MesosMessageBuilder) BuildContainerInfo(slaveID string, component *protocol.Application, reservedPorts []int32) (containerInfo *mesos.CommandInfo_ContainerInfo, portMapping []*protocol.PortMapping) {
	av := reservedPorts
	if component.GetDistribution() != protocol.Distribution_DOCKER {
		var pm []*protocol.PortMapping
		for _, port := range component.Ports {
			pm = append(pm, &protocol.PortMapping{
				Scheme:      proto.String(strings.ToUpper(port.GetKey())),
				PrivatePort: port.Value,
				PublicPort:  port.Value,
			})
		}
		return nil, pm
	}
	var options []string
	for _, port := range component.Ports {
		p, pp := av[0], av[1:]
		av = pp
		options = append(options, "-p", fmt.Sprintf("%v:%v", p, port.GetValue()))
		mapping := &protocol.PortMapping{
			Scheme:      proto.String(strings.ToUpper(port.GetKey())),
			PrivatePort: port.Value,
			PublicPort:  proto.Int32(p),
		}
		portMapping = append(portMapping, mapping)
	}
	containerInfo = &mesos.CommandInfo_ContainerInfo{
		Image:   proto.String("docker:///" + component.GetName() + ":" + component.GetVersion()),
		Options: options,
	}
	return containerInfo, portMapping
}

// BuildMesosCommand builds a mesos.CommandInfo object from a protocol.ApplicationComponent
// This is what drives our deployment and how it works.
func (b *MesosMessageBuilder) BuildMesosCommand(slaveID string, component *protocol.Application, reservedPorts []int32) (commandInfo *mesos.CommandInfo, portMapping []*protocol.PortMapping) {
	containerInfo, portMapping := b.BuildContainerInfo(slaveID, component, reservedPorts)
	commandInfo = &mesos.CommandInfo{
		Container:   containerInfo,
		Uris:        nil, // TODO: used to provide the docker image url for deimos?
		Environment: b.BuildTaskEnvironment(component.GetEnv(), component.GetPorts(), reservedPorts),
		Value:       component.Command,
		User:        nil, // TODO: allow this to be configured?
		HealthCheck: nil, // TODO: allow this to be configured?
	}
	return commandInfo, portMapping
}

// BuildTaskInfo builds a mesos.TaskInfo object from an offer and a scheduled component
func (b *MesosMessageBuilder) BuildTaskInfo(taskID string, offer *mesos.Offer, scheduled *protocol.ScheduledApp) (mesos.TaskInfo, []*protocol.PortMapping) {
	component := scheduled.App
	slaveID := offer.GetSlaveId().GetValue()

	takenRanges, reservedPorts := b.PortPicker.GetPorts(offer, len(component.GetPorts()))
	commandInfo, portMapping := b.BuildMesosCommand(slaveID, component, reservedPorts)

	taskInfo := mesos.TaskInfo{
		Name:      proto.String(scheduled.GetAppId()),
		TaskId:    &mesos.TaskID{Value: proto.String("exeggutor-task-" + taskID)},
		SlaveId:   offer.SlaveId,
		Command:   commandInfo,
		Resources: b.BuildResources(component, takenRanges),
		Executor:  nil, // TODO: Make use of an executor to increase visibility into execution
	}

	return taskInfo, portMapping
}
