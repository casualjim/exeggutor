package tasks

import (
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.tasks")

// TaskManager is an interface that abstracts over
// various implementations of task managers.
// This allows for task managers to be substituted in tests
// with simpler implementations
type TaskManager interface {
	exeggutor.Module

	SubmitApp(app []protocol.Application) error
	SaveApp(app *protocol.Application) error
	FulfillOffer(offer mesos.Offer) []mesos.TaskInfo

	TaskStopping(taskID *mesos.TaskID)
	TaskFailed(taskID *mesos.TaskID, slaveID *mesos.SlaveID)
	TaskFinished(taskID *mesos.TaskID, slaveID *mesos.SlaveID)
	TaskKilled(taskID *mesos.TaskID, slaveID *mesos.SlaveID)
	TaskLost(taskID *mesos.TaskID, slaveID *mesos.SlaveID)
	TaskRunning(taskID *mesos.TaskID, slaveID *mesos.SlaveID)
	TaskStaging(taskID *mesos.TaskID, slaveID *mesos.SlaveID)
	TaskStarting(taskID *mesos.TaskID, slaveID *mesos.SlaveID)

	FindTasksForApp(name string) ([]*mesos.TaskID, error)
	FindTasksForComponent(app, component string) ([]*mesos.TaskID, error)
	FindTaskForComponent(task string) (*mesos.TaskID, error)

	//RunningApps() []*protocol.Application
	// Schedules

	TasksToKill() <-chan *mesos.TaskID
}
