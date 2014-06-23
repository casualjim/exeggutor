package tasks

import (
	"fmt"
	"sync"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/queue"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
	"github.com/reverb/go-utils/flake"
)

// PortProvider an interface for generating ports
// It operates on a per slave id basis.
// The default implementation gets its range values from the
// config and keeps track of which slave has which port
// so that it never hands out a port
type PortProvider interface {
	Acquire(id string) (int, error)
	Release(id string, port int)
}

type defaultPortProvider struct {
	min      int
	max      int
	used     map[string][]int
	released map[string][]int
	current  map[string]int
	lock     *sync.Mutex
}

// NewPortProvider creates a new instance of the default port provider
func NewPortProvider(config *exeggutor.Config) PortProvider {
	return &defaultPortProvider{
		min:      config.FrameworkInfo.MinPort,
		max:      config.FrameworkInfo.MaxPort,
		used:     make(map[string][]int),
		released: make(map[string][]int),
		current:  make(map[string]int),
		lock:     &sync.Mutex{},
	}
}

func (p *defaultPortProvider) Acquire(slaveID string) (int, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	cur, ok := p.current[slaveID]

	if !ok {
		cur = p.min
		p.current[slaveID] = cur
		p.used[slaveID] = []int{cur}
		p.released[slaveID] = []int{}
		return cur, nil
	}

	// First try to see if we have a released port around for recycling
	next := p.popReleased(slaveID)
	if next == 0 {
		next = cur + 1
	}

	// paranoia! Keep incrementing the port number until we have one that's not in use
	// in practice this should normally return false or something must have gone horribly wrong
	for p.hasUsed(slaveID, next) {
		next++
	}

	// if we're at a port higher than the max we can't acquire new ports,
	// we've handed out 10k ports to that particular slave, it might also be
	// spontaneously exploding at this place or used to warm up a cold winter night.
	if p.overstepsMax(next) {
		return 0, fmt.Errorf("%v is larger than %v, can't acquire a new id for slave %s", next, p.max, slaveID)
	}

	// update the states for this newly acquired id
	p.current[slaveID] = next
	p.used[slaveID] = append(p.used[slaveID], next)

	return next, nil
}

func (p *defaultPortProvider) overstepsMax(port int) bool {
	return p.max < port
}

func (p *defaultPortProvider) hasUsed(slaveID string, port int) bool {
	inUse, ok := p.used[slaveID]
	if !ok {
		return false
	}

	for _, v := range inUse {
		if port == v {
			return true
		}
	}
	return false
}

func (p *defaultPortProvider) popReleased(slaveID string) int {
	available, ok := p.released[slaveID]
	if !ok || len(available) == 0 {
		return 0
	}

	popped, remaining := available[len(available)-1], available[:len(available)-1]
	p.released[slaveID] = remaining
	return popped
}

func (p *defaultPortProvider) Release(slaveID string, port int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var newUsed []int
	for _, v := range p.used[slaveID] {
		if v != port {
			newUsed = append(newUsed, v)
		}
	}
	p.used[slaveID] = newUsed
	p.released[slaveID] = append(p.released[slaveID], port)
}

// DefaultTaskManager the task manager accepts application manifests
// and schedules them when a suitable offer arrives.
// It then tracks the state of the running components, so that eventually
// an SLA enforcement service will be able to make sure the
// necessary application components are always alive.
type DefaultTaskManager struct {
	queue     TaskQueue
	taskStore store.KVStore
	config    *exeggutor.Config
	flake     queue.IDGenerator
	deploying map[string]*mesos.TaskInfo
	ports     PortProvider
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
		ports:     NewPortProvider(config),
	}, nil
}

// NewCustomDefaultTaskManager creates a new instance of a task manager with all the internal components injected
func NewCustomDefaultTaskManager(q TaskQueue, ts store.KVStore, config *exeggutor.Config, deploying map[string]*mesos.TaskInfo, ports PortProvider) (*DefaultTaskManager, error) {
	return &DefaultTaskManager{
		queue:     q,
		taskStore: ts,
		config:    config,
		flake:     flake.NewFlake(),
		deploying: deploying,
		ports:     ports,
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
	log.Debug("Submitting app: %+v", app)
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
	return BuildTaskInfo(taskID, &offer, scheduled, t.ports)
}

func (t *DefaultTaskManager) hasEnoughMem(availableMem float64, component *protocol.ApplicationComponent) bool {
	return availableMem >= float64(component.GetMem())
}

func (t *DefaultTaskManager) hasEnoughCPU(availableCpus float64, component *protocol.ApplicationComponent) bool {
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

	return t.hasEnoughCPU(availCpus, component.Component) && t.hasEnoughMem(availMem, component.Component)
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
	task := t.buildTaskInfo(offer, item)
	allQueued = append(allQueued, task)
	t.deploying[task.GetTaskId().GetValue()] = &task
	log.Debug("fullfilling offer with %+v", task)
	return allQueued
}

func (t *DefaultTaskManager) moveTaskToStore(taskID *mesos.TaskID) error {
	task, ok := t.deploying[taskID.GetValue()]
	if !ok {
		log.Warning("Couldn't find task with id: " + taskID.GetValue())
		return nil
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
