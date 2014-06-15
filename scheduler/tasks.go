package scheduler

import (
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/queue"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
)

// TaskManager the task manager accept
type TaskManager struct {
	queue          queue.Queue
	taskStore      store.KVStore
	config         *exeggutor.Config
	mesosScheduler *MesosScheduler
}

// NewTaskManager creates a new instance of a task manager with the values
// from the provided config.
func NewTaskManager(config *exeggutor.Config) (*TaskManager, error) {
	store, err := store.NewMdbStore(config.DataDirectory + "/tasks")
	if err != nil {
		return nil, err
	}

	return &TaskManager{
		queue:          queue.NewInMemoryQueue(),
		taskStore:      store,
		config:         config,
		mesosScheduler: NewMesosScheduler(config),
	}, nil
}

// Start starts the instance of the taks manager and all the components it depends on.
func (t *TaskManager) Start() error {
	err := t.taskStore.Start()
	if err != nil {
		return err
	}

	err = t.mesosScheduler.Start()
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
func (t *TaskManager) Stop() error {
	err1 := t.queue.Stop()
	err2 := t.mesosScheduler.Stop()
	err3 := t.taskStore.Stop()

	// This is a bit of a weird break down but this way
	// we preserve all error messages logged as warnings
	// even though we return the first one that failed
	// from this function
	if err1 != nil || err2 != nil || err3 != nil {
		log.Warning("There were problems shutting down the task manager:")
		if err1 != nil {
			log.Warning("%v", err1)
		}
		if err2 != nil {
			log.Warning("%v", err2)
		}
		if err3 != nil {
			log.Warning("%v", err3)
		}
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	if err3 != nil {
		return err3
	}
	return nil
}

// SubmitApp submits an application to the queue for scheduling on the
// cluster
func (t *TaskManager) SubmitApp(app protocol.ApplicationManifest) error {
	return t.queue.Enqueue(app)
}

// FrameworkID returns the framework id value if there is any already provided
func (t *TaskManager) FrameworkID() string {
	return t.mesosScheduler.FrameworkIDState.Get().GetValue()
}

// FulfillOffer tries to fullfil an offer with the biggest and oldest enqueued thing it can find.
// this can be an expensive operation when the queue is large, in practice this queue should never
// get very large because that would indicate we're grossly underprovisioned
// So when this starts taking too long we should provide more instances to this cluster
func (t *TaskManager) FulfillOffer(offer mesos.Offer) {

}
