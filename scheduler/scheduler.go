package scheduler

import (
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/state"
	"github.com/reverb/exeggutor/tasks"
	"github.com/reverb/go-utils/rvb_zk"

	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.scheduler")

var launched = false

// Framework is the object that listens to mesos resource offers and
// and tries to fullfil offers if it has applications queued for submission
type Framework struct {
	context *exeggutor.AppContext
	// FrameworkIDState the zookeeper backed framework id state for this application
	id            state.FrameworkIDState
	ownsFwIDState bool
	// Curator the zookeeper client library
	Curator     *rvb_zk.Curator
	ownsCurator bool
	// Driver the driver for the mesos framework
	driver      mesos.SchedulerDriver
	taskManager tasks.TaskManager
}

// NewFramework creates a new instance of Framework with the specified config
func NewFramework(context *exeggutor.AppContext, taskManager tasks.TaskManager) *Framework {
	log.Debug("Creating a new instance of a mesos scheduler")
	return &Framework{context: context, ownsCurator: true, ownsFwIDState: true, taskManager: taskManager}
}

// NewCustomFramework creates a new instance of a framework with all the dependencies injected
func NewCustomFramework(context *exeggutor.AppContext, fwID state.FrameworkIDState, curator *rvb_zk.Curator) *Framework {
	log.Debug("Creating a new custom instance of a mesos scheduler")
	return &Framework{context: context, id: fwID, Curator: curator}
}

// func NewCustomFrameworkWithScheduler(
// 	config *exeggutor.Config, fwID state.id, curator *rvb_zk.Curator, scheduler *mesos.Scheduler) *Framework {
// 	return nil
// }

func (fw *Framework) infoFromConfig() mesos.FrameworkInfo {
	return mesos.FrameworkInfo{
		User: proto.String(fw.context.Config.FrameworkInfo.User),
		Name: proto.String(fw.context.Config.FrameworkInfo.Name),
		Id:   fw.id.Get(),
	}
}

// SubmitApp submits an application to the queue for scheduling on the
// cluster
func (fw *Framework) SubmitApp(app []protocol.Application) error {
	return fw.taskManager.SubmitApp(app)
}

// KillApp stops all the components of an application
func (fw *Framework) KillApp(app string) error {
	taskIds, err := fw.taskManager.FindTasksForApp(app)
	if err != nil {
		return err
	}
	for _, taskID := range taskIds {
		err2 := fw.driver.KillTask(taskID)
		if err2 != nil {
			err = err2
		}
	}
	return err
}

// KillComponent stops a single component
func (fw *Framework) KillComponent(app, component string) error {
	taskIds, err := fw.taskManager.FindTasksForComponent(app, component)
	if err != nil {
		return err
	}
	for _, taskID := range taskIds {
		err2 := fw.driver.KillTask(taskID)
		if err2 != nil {
			err = err2
		}

	}
	return err
}

// KillComponentOnSlave stop s a single component instance on a particular slave
func (fw *Framework) KillComponentOnSlave(tID string) error {
	taskID, err := fw.taskManager.FindTaskForComponent(tID)
	if err != nil {
		return err
	}
	err = fw.driver.KillTask(taskID)
	return err
}

// ID gets the id of the framework is one is known for this framework at this stage.
func (fw *Framework) ID() string {
	if fw.id == nil || fw.id.Get() == nil {
		return ""
	}
	return fw.id.Get().GetValue()
}

func (fw *Framework) defaultMesosScheduler() *mesos.Scheduler {
	return &mesos.Scheduler{
		Registered: func(driver *mesos.SchedulerDriver, fwID mesos.FrameworkID, masterInfo mesos.MasterInfo) {
			log.Info("registered framework %v with master %v", fwID.GetValue(), masterInfo.GetId())
			fw.id.Set(&fwID)
		},
		OfferRescinded: func(driver *mesos.SchedulerDriver, offer mesos.OfferID) {
			log.Info("the offer %s was rescinded", offer.GetValue())
		},
		Disconnected: func(driver *mesos.SchedulerDriver) {
			log.Warning("Disconnected from master!")
			// TODO: terminate all the tasks
		},
		Reregistered: func(driver *mesos.SchedulerDriver, masterInfo mesos.MasterInfo) {
			log.Info("Re-registered with master %s:%d\n", masterInfo.GetHostname(), masterInfo.GetPort())
		},
		SlaveLost: func(driver *mesos.SchedulerDriver, slaveID mesos.SlaveID) {
			log.Warning("Lost slave", slaveID.GetValue())
			// TODO: terminate all the tasks belonging to this slave.
		},
		Error: func(driver *mesos.SchedulerDriver, message string) {
			log.Error("Got an error:", message)
		},
		StatusUpdate: func(driver *mesos.SchedulerDriver, status mesos.TaskStatus) {
			log.Info("Status update: %+v", status)
			taskID := status.GetTaskId().GetValue()
			slaveID := status.SlaveId.GetValue()
			switch status.GetState() {
			case mesos.TaskState_TASK_FAILED:
				log.Warning("Task %s failed on %s, because %s", taskID, slaveID, status.GetMessage())
				fw.taskManager.TaskFailed(status.GetTaskId(), status.SlaveId)
			case mesos.TaskState_TASK_FINISHED:
				log.Notice("Task %s finished on %s", taskID, slaveID)
				fw.taskManager.TaskFinished(status.GetTaskId(), status.SlaveId)
			case mesos.TaskState_TASK_KILLED:
				log.Warning("Task %s killed on %s, because %s", taskID, slaveID, status.GetMessage())
				fw.taskManager.TaskKilled(status.GetTaskId(), status.SlaveId)
			case mesos.TaskState_TASK_LOST:
				log.Warning("Task %s lost on %s, because %s", taskID, slaveID, status.GetMessage())
				fw.taskManager.TaskLost(status.GetTaskId(), status.SlaveId)
			case mesos.TaskState_TASK_RUNNING:
				log.Notice("Task %s running on %s", taskID, slaveID)
				fw.taskManager.TaskRunning(status.GetTaskId(), status.SlaveId)
			case mesos.TaskState_TASK_STAGING:
				log.Warning("Task %s is stuck in staging on %s, killing...", taskID, slaveID)
				fw.taskManager.TaskStaging(status.GetTaskId(), status.SlaveId)
				driver.KillTask(status.GetTaskId())
			case mesos.TaskState_TASK_STARTING:
				log.Warning("Task %s is stuck in starting on %s, killing...", taskID, slaveID)
				fw.taskManager.TaskStarting(status.GetTaskId(), status.SlaveId)
				driver.KillTask(status.GetTaskId())
			}

		},
		FrameworkMessage: func(driver *mesos.SchedulerDriver, executorID mesos.ExecutorID, slaveID mesos.SlaveID, data string) {
			log.Info("Got framework message from executor %s, slave %s, and data: %s\n", executorID.GetValue(), slaveID.GetValue(), data)
		},
		ExecutorLost: func(driver *mesos.SchedulerDriver, executorID mesos.ExecutorID, slaveID mesos.SlaveID, status int) {
			log.Error("Lost executor %s, on slave %s with status code %d\n", executorID.GetValue(), slaveID.GetValue(), status)
		},
		ResourceOffers: func(driver *mesos.SchedulerDriver, offers []mesos.Offer) {
			logged := false
			for _, offer := range offers {
				if fw.taskManager != nil {
					if !logged {
						log.Debug("Received %d offers:", len(offers))
						logged = true
					}
					log.Debug("  * %+v", offer)
					fulfilment := fw.taskManager.FulfillOffer(offer)
					if len(fulfilment) == 0 {
						driver.DeclineOffer(offer.GetId())
					} else {
						driver.LaunchTasks(offer.GetId(), fulfilment)
					}
				} else {
					log.Notice("Received an offer but no task manager is available to handle the offer, declining %s", offer.GetId().GetValue())
					driver.DeclineOffer(offer.GetId())
				}
			}
		},
	}
}

// Start initializes the scheduler and everything it depends on
func (fw *Framework) Start() error {
	uri := fw.context.Config.ZookeeperURL
	master := fw.context.Config.MesosMaster

	if fw.ownsCurator {
		curator, err := rvb_zk.NewCuratorFromURI(uri)
		if err != nil {
			log.Critical("Couldn't connect to zookeeper because %v", err)
			return err
		}
		fw.Curator = curator
	}

	if fw.ownsFwIDState {
		fw.id = state.NewZookeeperFrameworkIDState(fw.Curator.RootNode+"/framework/id", fw.Curator)
		fw.id.Start(true)
	}

	log.Debug("Creating mesos scheduler driver")
	fw.driver = mesos.SchedulerDriver{
		Master:    master,
		Framework: fw.infoFromConfig(),
		Scheduler: fw.defaultMesosScheduler(),
	}

	err := fw.driver.Init()
	if err != nil {
		log.Critical("Couldn't initialize the mesos scheduler driver, because %v", err)
		return err
	}
	err = fw.driver.Start()
	if err != nil {
		log.Critical("Couldn't start the mesos scheduler driver, because %v", err)
		return err
	}
	log.Notice("Started the exeggutor scheduler")
	return nil
}

// Stop stops the mesos scheduler driver
func (fw *Framework) Stop() error {

	err1 := fw.driver.Stop(false)
	fw.driver.Destroy()
	var err2 error
	if fw.ownsFwIDState {
		err2 = fw.id.Stop()
	}
	if fw.ownsCurator {
		fw.Curator.Close()
	}

	if err1 != nil || err2 != nil {
		log.Warning("Failed to stop the mesos scheduler, because:")
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

	log.Notice("Stopped the mesos scheduler and relevant stores")
	return nil
}
