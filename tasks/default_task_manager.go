package tasks

import (
	"time"

	"code.google.com/p/goprotobuf/proto"

	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/health"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/tasks/builders"
	task_queue "github.com/reverb/exeggutor/tasks/queue"
	task_store "github.com/reverb/exeggutor/tasks/store"
	"github.com/reverb/go-mesos/mesos"
)

// DefaultTaskManager the task manager accepts application manifests
// and schedules them when a suitable offer arrives.
// It then tracks the state of the running components, so that eventually
// an SLA enforcement service will be able to make sure the
// necessary application components are always alive.
type DefaultTaskManager struct {
	queue       task_queue.TaskQueue
	taskStore   task_store.TaskStore
	context     *exeggutor.AppContext
	builder     *builders.MesosMessageBuilder
	healtchecks health.HealthCheckScheduler
	closing     chan bool
}

// NewDefaultTaskManager creates a new instance of a task manager with the values
// from the provided config.
func NewDefaultTaskManager(context *exeggutor.AppContext) (*DefaultTaskManager, error) {
	store, err := task_store.New(context.Config)
	if err != nil {
		return nil, err
	}

	q := task_queue.New()
	return &DefaultTaskManager{
		queue:       q,
		taskStore:   store,
		context:     context,
		builder:     builders.New(context.Config),
		healtchecks: health.New(context),
		closing:     make(chan bool),
	}, nil
}

// // NewCustomDefaultTaskManager creates a new instance of a task manager with all the internal components injected
// func NewCustomDefaultTaskManager(q task_queue.TaskQueue, ts task_store.TaskStore, context *exeggutor.AppContext, builder *builders.MesosMessageBuilder) (*DefaultTaskManager, error) {
// 	return &DefaultTaskManager{
// 		queue:     q,
// 		taskStore: ts,
// 		context:   context,
// 		builder:   builder,
// 	}, nil
// }

// Start starts the instance of the taks manager and all the components it depends on.
func (t *DefaultTaskManager) Start() error {

	err := t.taskStore.Start()
	if err != nil {
		return err
	}

	err = t.queue.Start()
	if err != nil {
		// We failed to start, cleanup the task store initialization
		t.taskStore.Stop()
		return err
	}

	if t.healtchecks != nil {
		err = t.healtchecks.Start()
		if err != nil {
			// Stop these guys again we failed to start.
			t.taskStore.Stop()
			t.queue.Stop()
			return err
		}
	}

	go t.listenForHealthFailures()

	return nil
}

func (t *DefaultTaskManager) listenForHealthFailures() {
	for {
		select {
		case failure := <-t.healtchecks.Failures():
			log.Info("task %d failed the health check", failure.ID)
		case <-t.closing:
			return
		}
	}
}

// Stop stops this task manager, cleaning up any resources
// it might have required and owns.
func (t *DefaultTaskManager) Stop() error {
	var err3 error
	if t.healtchecks != nil {
		err3 = t.healtchecks.Stop()
	}

	err := t.taskStore.Stop()
	err2 := t.queue.Stop()

	if err != nil || err2 != nil || err3 != nil {
		log.Warning("There were problems shutting down the task manager:")
		if err != nil {
			log.Warning("%v", err)
		}
		if err2 != nil {
			log.Warning("%v", err2)
		}
		if err3 != nil {
			log.Warning("%v", err3)
		}
		if err != nil {
			return err
		}
		if err2 != nil {
			return err2
		}
		if err3 != nil {
			return err3
		}
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

// RunningApps finds all the tasks that are currently running
func (t *DefaultTaskManager) RunningApps(name string) ([]*mesos.TaskID, error) {
	return t.taskStore.FilterToTaskIds(func(item *protocol.DeployedAppComponent) bool {
		return item.GetAppName() == name && item.GetStatus() == protocol.AppStatus_STARTED
	})
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

func (t *DefaultTaskManager) buildTaskInfo(offer mesos.Offer, scheduled *protocol.ScheduledApp) (mesos.TaskInfo, []*protocol.PortMapping) {
	taskID, _ := t.context.IDGenerator.Next()
	return t.builder.BuildTaskInfo(taskID, &offer, scheduled)
}

func (t *DefaultTaskManager) fitsInOffer(offer mesos.Offer, component *protocol.ScheduledApp) bool {
	log.Debug("Checking that %+v fits in %+v", offer, component)
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

	hasEnoughCPU := availCpus >= float64(comp.GetCpus())
	hasEnoughMem := availMem >= float64(comp.GetMem())
	hasEnoughPorts := int(maxPortsLen) >= len(comp.GetPorts())
	log.Debug("the offer\ncpu: %t,\nmem: %t,\nports: %t", hasEnoughCPU, hasEnoughMem, hasEnoughPorts)

	return hasEnoughCPU && hasEnoughMem && hasEnoughPorts
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

	task, portMapping := t.buildTaskInfo(offer, item)
	deploying := &protocol.DeployedAppComponent{
		AppName:     item.AppName,
		Component:   item.App,
		TaskId:      task.GetTaskId(),
		Status:      protocol.AppStatus_DEPLOYING.Enum(),
		Slave:       task.GetSlaveId(),
		HostName:    offer.Hostname,
		PortMapping: portMapping,
		DeployedAt:  proto.Int64(time.Now().UnixNano() / 1000000),
	}
	err = t.taskStore.Save(deploying)
	if err != nil {
		return []mesos.TaskInfo{}
	}
	log.Debug("fullfilling offer with %+v", task)
	return []mesos.TaskInfo{task}
}

func (t *DefaultTaskManager) updateStatus(taskID *mesos.TaskID, status protocol.AppStatus) error {
	deploying, err := t.taskStore.Get(taskID.GetValue())
	if err != nil {
		return err
	}
	deploying.Status = status.Enum()
	if t.healtchecks != nil {
		if deploying.GetStatus() == protocol.AppStatus_STARTED {
			if err := t.healtchecks.Register(deploying); err != nil {
				log.Warning("Failed to unregister health check for %v, because %v", taskID.GetValue(), err)
			}
		} else {
			if err := t.healtchecks.Unregister(taskID); err != nil {
				log.Warning("Failed to unregister health check for %v, because %v", taskID.GetValue(), err)
			}
		}
	}
	if err := t.taskStore.Save(deploying); err != nil {
		log.Warning("Failed to unregister health check for %v, because %v", taskID.GetValue(), err)
	}
	return nil
}

func (t *DefaultTaskManager) forgetTask(taskID *mesos.TaskID) {
	if t.healtchecks != nil {
		if err := t.healtchecks.Unregister(taskID); err != nil {
			log.Warning("Failed to unregister health check for %v, because %v", taskID.GetValue(), err)
		}

	}
	if err := t.taskStore.Delete(taskID.GetValue()); err != nil {
		log.Warning("Failed to delete deployed app %v, because %v", taskID.GetValue(), err)
	}
}

// TaskStopping transitions this task into the stopping state
func (t *DefaultTaskManager) TaskStopping(taskID *mesos.TaskID) {
	t.updateStatus(taskID, protocol.AppStatus_STOPPING)
}

// TaskFailed a callback for when a task failed
func (t *DefaultTaskManager) TaskFailed(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Track failures and keep count, eventually alert
	t.updateStatus(taskID, protocol.AppStatus_STOPPED)
}

// TaskFinished a callback for when a task finishes successfully
func (t *DefaultTaskManager) TaskFinished(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Move task into finished state, delete in 30 days
	t.updateStatus(taskID, protocol.AppStatus_STOPPED)
}

// TaskKilled a callback for when a task is killed
func (t *DefaultTaskManager) TaskKilled(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// This is generally the tail end of a migration step
	t.updateStatus(taskID, protocol.AppStatus_STOPPED)
}

// TaskLost a callback for when a task was lost
func (t *DefaultTaskManager) TaskLost(taskID *mesos.TaskID, slaveID *mesos.SlaveID) {
	// Uh Oh I suppose we'd better reschedule this one ahead of everybody else
	t.updateStatus(taskID, protocol.AppStatus_STOPPED)
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
