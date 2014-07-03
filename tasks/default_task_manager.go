package tasks

import (
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"
	"github.com/reverb/go-utils/flake"
)

// DefaultTaskManager the task manager accepts application manifests
// and schedules them when a suitable offer arrives.
// It then tracks the state of the running components, so that eventually
// an SLA enforcement service will be able to make sure the
// necessary application components are always alive.
type DefaultTaskManager struct {
	queue     TaskQueue
	taskStore TaskStore
	config    *exeggutor.Config
	flake     exeggutor.IDGenerator
}

// NewDefaultTaskManager creates a new instance of a task manager with the values
// from the provided config.
func NewDefaultTaskManager(config *exeggutor.Config) (*DefaultTaskManager, error) {
	store, err := NewTaskStore(config)
	if err != nil {
		return nil, err
	}

	q := NewTaskQueue()
	return &DefaultTaskManager{
		queue:     q,
		taskStore: store,
		config:    config,
		flake:     flake.NewFlake(),
	}, nil
}

// NewCustomDefaultTaskManager creates a new instance of a task manager with all the internal components injected
func NewCustomDefaultTaskManager(q TaskQueue, ts TaskStore, config *exeggutor.Config) (*DefaultTaskManager, error) {
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
func (t *DefaultTaskManager) SubmitApp(app []protocol.Application) error {
	log.Debug("Submitting app: %+v", app)
	for _, comp := range app {
		component := protocol.ScheduledApp{
			Name:    comp.Name,
			AppName: comp.AppName,
			App:     &comp,
		}
		t.queue.Enqueue(&component)
	}
	return nil
}

// FindTasksForApp finds all the tasks for the specified application name
func (t *DefaultTaskManager) FindTasksForApp(name string) ([]*mesos.TaskID, error) {
	return t.taskStore.FilterToTaskIds(func(item *protocol.DeployedAppComponent) bool {
		return item.GetAppName() == name
	})
}

// FindTasksForComponent finds all the deployed tasks for the specified component
func (t *DefaultTaskManager) FindTasksForComponent(app, component string) ([]*mesos.TaskID, error) {
	return t.taskStore.FilterToTaskIds(func(item *protocol.DeployedAppComponent) bool {
		return item.GetAppName() == app && item.Component != nil && item.Component.GetName() == component
	})
}

// FindTaskForComponent finds the task for the specified task (single instance)
func (t *DefaultTaskManager) FindTaskForComponent(task string) (*mesos.TaskID, error) {
	res, err := t.taskStore.Get(task)
	if err != nil || res == nil {
		return nil, err
	}
	return res.TaskId, nil
}

func (t *DefaultTaskManager) buildTaskInfo(offer mesos.Offer, scheduled *protocol.ScheduledApp) mesos.TaskInfo {
	taskID, _ := t.flake.Next()
	return BuildTaskInfo(taskID, &offer, scheduled)
}

func (t *DefaultTaskManager) fitsInOffer(offer mesos.Offer, component *protocol.ScheduledApp) bool {
	var availCpus float64
	var availMem float64
	var maxPortsLen uint64

	for _, resource := range offer.Resources {
		switch resource.GetName() {
		case "cpus":
			availCpus = availCpus + resource.GetScalar().GetValue()
		case "mem":
			availMem = availMem + resource.GetScalar().GetValue()
		case "ports":
			var max uint64
			for _, r := range resource.GetRanges().GetRange() {
				numAvail := r.GetEnd() - r.GetBegin() + 1
				if max < numAvail {
					max = numAvail
				}
			}
			maxPortsLen = max
		}
	}

	comp := component.App

	return availCpus >= float64(comp.GetCpus()) && // has enough cpu
		availMem >= float64(comp.GetMem()) && // has enough memory
		int(maxPortsLen) >= len(comp.GetPorts()) // has enough consecutive free ports
}

// FulfillOffer tries to fullfil an offer with the biggest and oldest enqueued thing it can find.
// this can be an expensive operation when the queue is large, in practice this queue should never
// get very large because that would indicate we're grossly underprovisioned
// So when this starts taking too long we should provide more instances to this cluster
func (t *DefaultTaskManager) FulfillOffer(offer mesos.Offer) []mesos.TaskInfo {

	thatFits := func(i *protocol.ScheduledApp) bool { return t.fitsInOffer(offer, i) }

	item, err := t.queue.DequeueFirst(thatFits)
	if err != nil {
		log.Critical("Couldn't dequeue from the task queue because: %v", err)
		return nil
	}

	if item == nil {
		log.Debug("Couldn't get an item of the queue, skipping this one")
		return nil
	}

	task := t.buildTaskInfo(offer, item)
	allQueued := []mesos.TaskInfo{task}
	status := protocol.AppStatus_DEPLOYING
	deploying := &protocol.DeployedAppComponent{
		AppName:   item.AppName,
		Component: item.App,
		TaskId:    task.GetTaskId(),
		Status:    &status,
		Slave:     task.GetSlaveId(),
	}
	t.taskStore.Save(deploying)

	log.Debug("fullfilling offer with %+v", task)
	return allQueued
}

func (t *DefaultTaskManager) updateStatus(taskID *mesos.TaskID, status protocol.AppStatus) error {
	deploying, err := t.taskStore.Get(taskID.GetValue())
	if err != nil {
		return err
	}
	deploying.Status = &status
	if err := t.taskStore.Save(deploying); err != nil {
		return err
	}
	return nil
}

func (t *DefaultTaskManager) forgetTask(taskID *mesos.TaskID) {
	// delete(t.deploying, taskID.GetValue())
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
	err := t.updateStatus(taskID, protocol.AppStatus_STARTED)
	if err != nil {
		log.Error("%v", err)
	}
}

// TaskStaging a callback for when a task that doesn't belong to this framework ends up here anyway
// we should remove traces of it should it still exist.
func (t *DefaultTaskManager) TaskStaging(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// We scheduled this app for deployment but nothing else happened, this is like an ack of the scheduler
	t.forgetTask(taskID)
}

// TaskStarting a callback for when a task transitions from staging to starting (is being deployed)
func (t *DefaultTaskManager) TaskStarting(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// We made it to a slave and the deployment process has begun
	t.forgetTask(taskID)
}
