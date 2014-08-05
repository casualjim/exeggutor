package tasks

import (
	"time"

	"code.google.com/p/goprotobuf/proto"

	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/health"
	"github.com/reverb/exeggutor/health/sla"
	"github.com/reverb/exeggutor/protocol"
	app_store "github.com/reverb/exeggutor/store/apps"
	task_store "github.com/reverb/exeggutor/store/tasks"
	"github.com/reverb/exeggutor/tasks/builders"
	task_queue "github.com/reverb/exeggutor/tasks/queue"
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
	appStore    app_store.AppStore
	context     *exeggutor.AppContext
	builder     *builders.MesosMessageBuilder
	healtchecks health.HealthCheckScheduler
	slaMonitor  sla.SLAMonitor
	closing     chan chan bool
	tasksToKill chan *mesos.TaskID
}

// NewDefaultTaskManager creates a new instance of a task manager with the values
// from the provided config.
func NewDefaultTaskManager(context *exeggutor.AppContext, appStore app_store.AppStore) (*DefaultTaskManager, error) {
	store, err := task_store.New(context.Config)
	if err != nil {
		return nil, err
	}

	//appStore := context.AppStore
	// if err != nil {
	// 	return nil, err
	// }

	q := task_queue.New()
	return &DefaultTaskManager{
		queue:       q,
		taskStore:   store,
		appStore:    appStore,
		context:     context,
		builder:     builders.New(context.Config),
		healtchecks: health.New(context),
		slaMonitor:  sla.New(store, appStore, q),
		closing:     make(chan chan bool),
		tasksToKill: make(chan *mesos.TaskID),
	}, nil
}

// TasksToKill a channel over which tasks that should be killed are received
func (t *DefaultTaskManager) TasksToKill() <-chan *mesos.TaskID {
	return t.tasksToKill
}

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

	err = t.slaMonitor.Start()
	if err != nil {
		// We failed to start, cleanup the queue and taskstore too
		t.taskStore.Stop()
		t.queue.Stop()
		return err
	}

	if t.healtchecks != nil {
		err = t.healtchecks.Start()
		if err != nil {
			// Stop these guys again we failed to start.
			t.taskStore.Stop()
			t.queue.Stop()
			// t.slaMonitor.Stop()
			return err
		}
		go t.listenForHealthFailures()
	}

	return nil
}

func (t *DefaultTaskManager) listenForHealthFailures() {
	for {
		select {
		case failure := <-t.healtchecks.Failures():
			log.Info("task %d failed the health check", failure.ID)
			deployment, err := t.taskStore.Get(failure.ID)
			if err != nil {
				log.Error("Failed to get a deployment for marking as failure, because: %v", err)
			}
			if deployment != nil && err == nil {
				t.updateStatus(deployment.GetTaskId(), protocol.AppStatus_UNHEALTHY)
				t.tasksToKill <- deployment.GetTaskId()
			}

		case scaleReq := <-t.slaMonitor.ScaleUpOrDown():
			// We ignore requests where the count is 0
			if scaleReq.Count > 0 {

			} else if scaleReq.Count < 0 {

			}

		case closed := <-t.closing:
			// We stop healthchecks and slaMonitor here so that this loop
			// doesn't start doing weird things.
			if err := t.healtchecks.Stop(); err != nil {
				log.Warning("There was an error closing the health checks", err)
			}
			if err2 := t.slaMonitor.Stop(); err2 != nil {
				log.Warning("There was an error closing the SLA monitor", err2)
			}
			closed <- true
			return
		}
	}
}

// Stop stops this task manager, cleaning up any resources
// it might have required and owns.
func (t *DefaultTaskManager) Stop() error {
	// stop listening for things first
	boolc := make(chan bool)
	t.closing <- boolc
	<-boolc

	err := t.taskStore.Stop()
	err2 := t.queue.Stop()

	if err != nil || err2 != nil {
		log.Warning("There were problems shutting down the task manager:")
		if err != nil {
			log.Warning("%v", err)
		}
		if err2 != nil {
			log.Warning("%v", err2)
		}
		if err != nil {
			return err
		}
		if err2 != nil {
			return err2
		}
	}
	close(t.tasksToKill)
	close(t.closing)
	close(boolc)

	return nil
}

// SaveApp saves an application
func (t *DefaultTaskManager) SaveApp(app *protocol.Application) error {
	log.Debug("Saving app: %+v", app)
	return t.appStore.Save(app)
}

// SubmitApp submits an application to the queue for scheduling on the
// cluster
func (t *DefaultTaskManager) SubmitApp(app []protocol.Application) error {
	log.Debug("Submitting app: %+v", app)
	for _, comp := range app {
		t.scheduleAppForDeployment(&comp)
	}
	return nil
}

func (t *DefaultTaskManager) scheduleAppForDeployment(app *protocol.Application) {
	log.Debug("Enqueueing for deployment with more instances (%t) %+v", t.slaMonitor.CanDeployMoreInstances(app), app)
	if !t.slaMonitor.CanDeployMoreInstances(app) {
		log.Warning("Can't deploy another instance of %s, the max instances have been reached")
		return
	}
	log.Debug("We can deploy more instances of %+v", app)
	component := protocol.ScheduledApp{
		AppId: app.Id,
		App:   app,
	}
	t.queue.Enqueue(&component)
}

// RunningApps finds all the tasks that are currently running
func (t *DefaultTaskManager) RunningApps(id string) (tasks []*mesos.TaskID, err error) {
	t.taskStore.ForEach(func(item *protocol.Deployment) {
		app, err := t.appStore.Get(item.GetAppId())
		if err != nil {
			log.Warning("Couldn't get the application %s linked to the task id %s, because: %v", item.GetAppId(), item.GetTaskId().GetValue(), err)
		}
		if app != nil && app.GetAppName() == id && item.GetStatus() == protocol.AppStatus_STARTED {
			tasks = append(tasks, item.GetTaskId())
		}
	})
	return
}

// FindTasksForApp finds all the tasks for the specified application name
func (t *DefaultTaskManager) FindTasksForApp(name string) (tasks []*mesos.TaskID, err error) {
	t.taskStore.ForEach(func(item *protocol.Deployment) {
		app, err := t.appStore.Get(item.GetAppId())
		if err != nil {
			log.Warning("Couldn't get the application %s linked to the task id %s, because: %v", item.GetAppId(), item.GetTaskId().GetValue(), err)
		}
		if app != nil && app.GetAppName() == name {
			tasks = append(tasks, item.GetTaskId())
		}
	})
	return
}

// FindTasksForComponent finds all the deployed tasks for the specified component
func (t *DefaultTaskManager) FindTasksForComponent(appName, component string) (tasks []*mesos.TaskID, err error) {
	t.taskStore.ForEach(func(item *protocol.Deployment) {
		app, err := t.appStore.Get(item.GetAppId())
		if err != nil {
			log.Warning("Couldn't get the application %s linked to the task id %s, because: %v", item.GetAppId(), item.GetTaskId().GetValue(), err)
		}
		if app != nil && app.GetAppName() == appName && app.GetName() == component {
			tasks = append(tasks, item.GetTaskId())
		}
	})
	return
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

	var item *protocol.ScheduledApp
	seen := false
	// dequeue items that fit but are saturated until we get one that fits and isn't saturated
	// in theory this should not occur because we've got this guard at enqueue time too.
	for item == nil && !seen && t.slaMonitor.CanDeployMoreInstances(item.GetApp()) {
		// for item == nil && !seen {
		log.Debug("Checking queue: %+v", t.queue)
		i, err := t.queue.DequeueFirst(thatFits)
		if err != nil {
			log.Critical("Couldn't dequeue from the task queue because: %v", err)
			return nil
		}
		item = i
		seen = true
	}
	if item == nil {
		log.Debug("Couldn't get an item of the queue, skipping this one")
		return nil
	}

	task, portMapping := t.buildTaskInfo(offer, item)
	deploying := &protocol.Deployment{
		AppId:       proto.String(item.GetAppId()),
		TaskId:      task.GetTaskId(),
		Status:      protocol.AppStatus_DEPLOYING.Enum(),
		Slave:       task.GetSlaveId(),
		HostName:    offer.Hostname,
		PortMapping: portMapping,
		DeployedAt:  proto.Int64(time.Now().UnixNano() / 1000000),
	}
	err := t.taskStore.Save(deploying)
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

	if err := t.taskStore.Save(deploying); err != nil {
		log.Warning("Failed to save task %v, because %v", taskID.GetValue(), err)
		return err
	}

	log.Debug("Getting from appstore %v", deploying)
	app, err := t.appStore.Get(deploying.GetAppId())
	if err != nil {
		log.Warning("Failed to retrieve application %v, because %v", deploying.GetAppId(), err)
	}

	if app != nil {
		if t.slaMonitor.NeedsMoreInstances(app) {
			t.scheduleAppForDeployment(app)
		}
		if t.healtchecks != nil {
			if deploying.GetStatus() == protocol.AppStatus_STARTED {
				if err := t.healtchecks.Register(deploying, app); err != nil {
					log.Warning("Failed to unregister health check for %v, because %v", taskID.GetValue(), err)
				}
			} else {
				if err := t.healtchecks.Unregister(taskID); err != nil {
					log.Warning("Failed to unregister health check for %v, because %v", taskID.GetValue(), err)
				}
			}
		}
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
	t.updateStatus(taskID, protocol.AppStatus_FAILED)
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
	t.updateStatus(taskID, protocol.AppStatus_FAILED)
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
