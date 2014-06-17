package tasks

import (
	"container/heap"

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
	queue     *TaskQueue
	taskStore store.KVStore
	config    *exeggutor.Config
	flake     queue.IDGenerator
}

// NewDefaultTaskManager creates a new instance of a task manager with the values
// from the provided config.
func NewDefaultTaskManager(config *exeggutor.Config) (*DefaultTaskManager, error) {
	store, err := store.NewMdbStore(config.DataDirectory + "/tasks")
	// TODO: replace this untyped queue with one that only accepts the
	//       right types. this is a bit dangerous and requires contextual knowledge
	//       or results in surprises at runtime.
	//q, err := queue.NewMdbQueue(config.DataDirectory+"/queues/tasks", &ScheduledAppComponentSerializer{})
	// q := queue.NewInMemoryQueue()
	if err != nil {
		return nil, err
	}

	q := &TaskQueue{}
	heap.Init(q)
	return &DefaultTaskManager{
		queue:     q,
		taskStore: store,
		config:    config,
		flake:     flake.NewFlake(),
		//mesosFramework: NewFramework(config),
	}, nil
}

// NewCustomDefaultTaskManager creates a new instance of a task manager with all the internal components injected
func NewCustomDefaultTaskManager(q *TaskQueue, ts store.KVStore, config *exeggutor.Config) (*DefaultTaskManager, error) {
	heap.Init(q)
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

	//err = t.queue.Start()
	//if err != nil {
	//return err
	//}

	return nil
}

// Stop stops this task manager, cleaning up any resources
// it might have required and owns.
func (t *DefaultTaskManager) Stop() error {
	//err1 := t.queue.Stop()
	err1 := t.taskStore.Stop()

	// This is a bit of a weird break down but this way
	// we preserve all error messages logged as warnings
	// even though we return the first one that failed
	// from this function
	if err1 != nil {
		log.Warning("There were problems shutting down the task manager:")
		log.Warning("%v", err1)
		return err1
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
		heap.Push(t.queue, &component)
	}
	return nil
}

func (t *DefaultTaskManager) buildTaskInfo(offer mesos.Offer, scheduled *protocol.ScheduledAppComponent) mesos.TaskInfo {
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

func (t *DefaultTaskManager) hasEnoughMem(availableMem float64, component *protocol.ApplicationComponent) bool {
	return availableMem >= float64(component.GetMem())
}

func (t *DefaultTaskManager) hasEnoughCpu(availableCpus float64, component *protocol.ApplicationComponent) bool {
	return availableCpus >= float64(component.GetCpus())
}

func (t *DefaultTaskManager) fitsInOffer(offer mesos.Offer, component *protocol.ScheduledAppComponent) bool {
	var availCpus float64
	var availMem float64

	for _, resource := range offer.Resources {
		switch resource.GetName() {
		case "cpus":
			availCpus = resource.GetScalar().GetValue()
		case "mem":
			availMem = resource.GetScalar().GetValue()
		}
	}

	return t.hasEnoughCpu(availCpus, component.Component) && t.hasEnoughMem(availMem, component.Component)
}

// FulfillOffer tries to fullfil an offer with the biggest and oldest enqueued thing it can find.
// this can be an expensive operation when the queue is large, in practice this queue should never
// get very large because that would indicate we're grossly underprovisioned
// So when this starts taking too long we should provide more instances to this cluster
func (t *DefaultTaskManager) FulfillOffer(offer mesos.Offer) []mesos.TaskInfo {
	var allQueued []mesos.TaskInfo
	queue := *t.queue
	for _, scheduled := range queue {
		if t.fitsInOffer(offer, scheduled) {
			allQueued = append(allQueued, t.buildTaskInfo(offer, scheduled))
			heap.Remove(t.queue, int(scheduled.GetPosition()))
		}
	}
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
