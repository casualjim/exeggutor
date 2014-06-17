package tasks

import (
	"errors"
	"strconv"
	"strings"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/queue"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
	"github.com/reverb/go-utils/flake"
)

// DefaultTaskManager the task manager accept
type DefaultTaskManager struct {
	queue     queue.Queue
	taskStore store.KVStore
	config    *exeggutor.Config
	flake     queue.IDGenerator
}

// ScheduledAppComponentSerializer a struct to implement a serializer for an
// exeggutor.protocol.ScheduledAppComponent
type ScheduledAppComponentSerializer struct{}

// ReadBytes ScheduledAppComponentSerializer a byte slice into a ScheduledAppComponent
func (a *ScheduledAppComponentSerializer) ReadBytes(data []byte) (interface{}, error) {
	appManifest := &protocol.ScheduledAppComponent{}
	err := proto.Unmarshal(data, appManifest)
	if err != nil {
		return nil, err
	}
	return appManifest, nil
}

// WriteBytes converts an application manifest into a byte slice
func (a *ScheduledAppComponentSerializer) WriteBytes(target interface{}) ([]byte, error) {
	msg, ok := target.(proto.Message)
	if !ok {
		return nil, errors.New("Expected to serialize an application manifest which is a proto message.")
	}
	return proto.Marshal(msg)
}

// NewDefaultTaskManager creates a new instance of a task manager with the values
// from the provided config.
func NewDefaultTaskManager(config *exeggutor.Config) (*DefaultTaskManager, error) {
	store, err := store.NewMdbStore(config.DataDirectory + "/tasks")
	// TODO: replace this untyped queue with one that only accepts the
	//       right types. this is a bit dangerous and requires contextual knowledge
	//       or results in surprises at runtime.
	q, err := queue.NewMdbQueue(config.DataDirectory+"/queues/tasks", &ScheduledAppComponentSerializer{})
	// q := queue.NewInMemoryQueue()
	if err != nil {
		return nil, err
	}

	return &DefaultTaskManager{
		queue:     q,
		taskStore: store,
		config:    config,
		flake:     flake.NewFlake(),
		//mesosFramework: NewFramework(config),
	}, nil
}

// NewCustomDefaultTaskManager creates a new instance of a task manager with all the internal components injected
func NewCustomDefaultTaskManager(q queue.Queue, ts store.KVStore, config *exeggutor.Config) (*DefaultTaskManager, error) {
	return &DefaultTaskManager{
		queue:     q,
		taskStore: ts,
		config:    config,
		flake:     flake.NewFlake(),
	}, nil
}

// Start starts the instance of the taks manager and all the components it depends on.
func (t *DefaultTaskManager) Start() error {
	err := t.taskStore.Start()
	if err != nil {
		return err
	}

	//err = t.mesosFramework.Start()
	//if err != nil {
	//return err
	//}

	err = t.queue.Start()
	if err != nil {
		return err
	}

	return nil
}

// Stop stops this task manager, cleaning up any resources
// it might have required and owns.
func (t *DefaultTaskManager) Stop() error {
	err1 := t.queue.Stop()
	err2 := t.taskStore.Stop()

	// This is a bit of a weird break down but this way
	// we preserve all error messages logged as warnings
	// even though we return the first one that failed
	// from this function
	if err1 != nil || err2 != nil {
		log.Warning("There were problems shutting down the task manager:")
		if err1 != nil {
			log.Warning("%v", err1)
		}
		if err2 != nil {
			log.Warning("%v", err2)
		}
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

// SubmitApp submits an application to the queue for scheduling on the
// cluster
func (t *DefaultTaskManager) SubmitApp(app protocol.ApplicationManifest) error {
	for _, comp := range app.Components {
		component := protocol.ScheduledAppComponent{
			Name:      comp.Name,
			AppName:   app.Name,
			Component: comp,
		}
		if err := t.queue.Enqueue(component); err != nil {
			return err
		}
	}
	return nil
}

func BuildResources(component *protocol.ApplicationComponent) []*mesos.Resource {
	return []*mesos.Resource{
		mesos.ScalarResource("cpus", float64(component.GetCpus())),
		mesos.ScalarResource("mem", float64(component.GetMem())),
	}
}

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

func (t *DefaultTaskManager) buildTaskInfo(offer mesos.Offer, scheduled protocol.ScheduledAppComponent) mesos.TaskInfo {
	taskID, _ := t.flake.Next()
	component := scheduled.Component
	info := mesos.TaskInfo{
		Name:      scheduled.Name,
		TaskId:    &mesos.TaskID{Value: proto.String("exeggutor-task-" + taskID)},
		SlaveId:   offer.SlaveId,
		Command:   BuildMesosCommand(component),
		Resources: BuildResources(component),
		Executor:  nil, // TODO: Make use of an executor to increase visibility into execution
	}
	return info
}

// FulfillOffer tries to fullfil an offer with the biggest and oldest enqueued thing it can find.
// this can be an expensive operation when the queue is large, in practice this queue should never
// get very large because that would indicate we're grossly underprovisioned
// So when this starts taking too long we should provide more instances to this cluster
func (t *DefaultTaskManager) FulfillOffer(offer mesos.Offer) []mesos.TaskInfo {
	var allQueued []mesos.TaskInfo
	t.queue.ForEach(func(v interface{}) {
		scheduled := v.(protocol.ScheduledAppComponent)
		allQueued = append(allQueued, t.buildTaskInfo(offer, scheduled))
	})
	return allQueued
}

// TaskFailed a callback for when a task failed
func (t *DefaultTaskManager) TaskFailed(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Track failures and keep count, eventually alert
}

// TaskFinished a callback for when a task finishes successfully
func (t *DefaultTaskManager) TaskFinished(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Move task into finished state, delete in 30 days
}

// TaskKilled a callback for when a task is killed
func (t *DefaultTaskManager) TaskKilled(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// This is generally the tail end of a migration step
}

// TaskLost a callback for when a task was lost
func (t *DefaultTaskManager) TaskLost(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Uh Oh I suppose we'd better reschedule this one ahead of everybody else
}

// TaskRunning a callback for when a task enters the running state
func (t *DefaultTaskManager) TaskRunning(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// All is well put this task in the running state in the UI
}

// TaskStaging a callback for when a task enters the first stage (probably never occurs in a framework)
func (t *DefaultTaskManager) TaskStaging(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// We scheduled this app for deployment but nothing else happened, this is like an ack of the scheduler
}

// TaskStarting a callback for when a task transitions from staging to starting (is being deployed)
func (t *DefaultTaskManager) TaskStarting(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// We made it to a slave and the deployment process has begun
}
