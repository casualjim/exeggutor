package tasks

import (
	"errors"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/queue"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
)

// DefaultTaskManager the task manager accept
type DefaultTaskManager struct {
	queue     queue.Queue
	taskStore store.KVStore
	config    *exeggutor.Config
}

// ApplicationManifestSerializer a struct to implement a serializer for an
// exeggutor.protocol.ApplicationManifest
type ApplicationManifestSerializer struct{}

// ReadBytes converts a byte slice into an ApplicationManifest
func (a *ApplicationManifestSerializer) ReadBytes(data []byte) (interface{}, error) {
	appManifest := &protocol.ApplicationManifest{}
	err := proto.Unmarshal(data, appManifest)
	if err != nil {
		return nil, err
	}
	return appManifest, nil
}

// WriteBytes converts an application manifest into a byte slice
func (a *ApplicationManifestSerializer) WriteBytes(target interface{}) ([]byte, error) {
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
	q, err := queue.NewMdbQueue(config.DataDirectory+"/queues/tasks", &ApplicationManifestSerializer{})
	// q := queue.NewInMemoryQueue()
	if err != nil {
		return nil, err
	}

	return &DefaultTaskManager{
		queue:     q,
		taskStore: store,
		config:    config,
		//mesosFramework: NewFramework(config),
	}, nil
}

// NewCustomDefaultTaskManager creates a new instance of a task manager with all the internal components injected
func NewCustomDefaultTaskManager(q queue.Queue, ts store.KVStore, config *exeggutor.Config) (*DefaultTaskManager, error) {
	return &DefaultTaskManager{
		queue:     q,
		taskStore: ts,
		config:    config,
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
	return t.queue.Enqueue(app)
}

// FulfillOffer tries to fullfil an offer with the biggest and oldest enqueued thing it can find.
// this can be an expensive operation when the queue is large, in practice this queue should never
// get very large because that would indicate we're grossly underprovisioned
// So when this starts taking too long we should provide more instances to this cluster
func (t *DefaultTaskManager) FulfillOffer(offer mesos.Offer) []mesos.TaskInfo {
	return []mesos.TaskInfo{}
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
