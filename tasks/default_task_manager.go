package tasks

import (
	"errors"

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
	queue     TaskQueue
	taskStore store.KVStore
	config    *exeggutor.Config
	flake     queue.IDGenerator
	deploying map[string]*mesos.TaskInfo
}

// NewDefaultTaskManager creates a new instance of a task manager with the values
// from the provided config.
func NewDefaultTaskManager(config *exeggutor.Config) (*DefaultTaskManager, error) {
	store, err := store.NewMdbStore(config.DataDirectory + "/tasks")
	if err != nil {
		return nil, err
	}

	q := NewTaskQueue()
	return &DefaultTaskManager{
		queue:     q,
		taskStore: store,
		config:    config,
		flake:     flake.NewFlake(),
		deploying: make(map[string]*mesos.TaskInfo),
	}, nil
}

// NewCustomDefaultTaskManager creates a new instance of a task manager with all the internal components injected
func NewCustomDefaultTaskManager(q TaskQueue, ts store.KVStore, config *exeggutor.Config, deploying map[string]*mesos.TaskInfo) (*DefaultTaskManager, error) {
	return &DefaultTaskManager{
		queue:     q,
		taskStore: ts,
		config:    config,
		flake:     flake.NewFlake(),
		deploying: deploying,
	}, nil
}

// Start starts the instance of the taks manager and all the components it depends on.
func (t *DefaultTaskManager) Start() error {
	err := t.taskStore.Start()
	if err != nil {
		return err
	}

	err = t.queue.Start()
	if err != nil {
		return err
	}

	return nil
}

// Stop stops this task manager, cleaning up any resources
// it might have required and owns.
func (t *DefaultTaskManager) Stop() error {
	err := t.taskStore.Stop()
	err2 := t.queue.Stop()
	if err != nil {
		log.Warning("There were problems shutting down the task manager:")
		log.Warning("%v", err)
		if err2 != nil {
			log.Warning("%v", err2)
		}
		return err
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
		t.queue.Enqueue(&component)
	}
	return nil
}

func (t *DefaultTaskManager) buildTaskInfo(offer mesos.Offer, scheduled *protocol.ScheduledAppComponent) mesos.TaskInfo {
	taskID, _ := t.flake.Next()
	return BuildTaskInfo(taskID, &offer, scheduled)
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
	item, err := t.queue.DequeueFirst(func(i *protocol.ScheduledAppComponent) bool { return t.fitsInOffer(offer, i) })
	if err != nil {
		log.Critical("Couldn't dequeue from the task queue because: %v", err)
		return nil
	}

	if item == nil {
		log.Debug("Couldn't get an item of the queue, skipping this one")
		return nil
	}
	allQueued = append(allQueued, t.buildTaskInfo(offer, item))
	return allQueued
}

func (t *DefaultTaskManager) moveTaskToStore(taskID *mesos.TaskID) error {
	task, ok := t.deploying[taskID.GetValue()]
	if !ok {
		return errors.New("Couldn't find task with id: " + taskID.GetValue())
	}
	bytes, err := proto.Marshal(task)
	if err != nil {
		return err
	}
	if err := t.taskStore.Set(task.GetTaskId().GetValue(), bytes); err != nil {
		return err
	}
	delete(t.deploying, taskID.GetValue())
	return nil
}

func (t *DefaultTaskManager) forgetTask(taskID *mesos.TaskID) {
	delete(t.deploying, taskID.GetValue())
	err := t.taskStore.Delete(taskID.GetValue())
	if err != nil {
		log.Error("%v", err)
	}
}

// TaskFailed a callback for when a task failed
func (t *DefaultTaskManager) TaskFailed(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Track failures and keep count, eventually alert
	t.forgetTask(taskID)
}

// TaskFinished a callback for when a task finishes successfully
func (t *DefaultTaskManager) TaskFinished(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Move task into finished state, delete in 30 days
	t.forgetTask(taskID)
}

// TaskKilled a callback for when a task is killed
func (t *DefaultTaskManager) TaskKilled(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// This is generally the tail end of a migration step
	t.forgetTask(taskID)
}

// TaskLost a callback for when a task was lost
func (t *DefaultTaskManager) TaskLost(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Uh Oh I suppose we'd better reschedule this one ahead of everybody else
	t.forgetTask(taskID)
}

// TaskRunning a callback for when a task enters the running state
func (t *DefaultTaskManager) TaskRunning(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// All is well put this task in the running state in the UI
	err := t.moveTaskToStore(taskID)
	if err != nil {
		log.Error("%v", err)
	}
}

// TaskStaging a callback for when a task enters the first stage (probably never occurs in a framework)
func (t *DefaultTaskManager) TaskStaging(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// We scheduled this app for deployment but nothing else happened, this is like an ack of the scheduler
	err := t.moveTaskToStore(taskID)
	if err != nil {
		log.Error("%v", err)
	}
}

// TaskStarting a callback for when a task transitions from staging to starting (is being deployed)
func (t *DefaultTaskManager) TaskStarting(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// We made it to a slave and the deployment process has begun
	err := t.moveTaskToStore(taskID)
	if err != nil {
		log.Error("%v", err)
	}
}
