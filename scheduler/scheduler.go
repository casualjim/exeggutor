package scheduler

import (
	"time"

	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/state"
	"github.com/reverb/go-utils/flake"
	"github.com/reverb/go-utils/rvb_zk"
	"github.com/samuel/go-zookeeper/zk"

	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.scheduler")

var launched = true

// MesosScheduler is the object that listens to mesos resource offers and
// and tries to fullfil offers if it has applications queued for submission
type MesosScheduler struct {
	config *exeggutor.Config
	// FrameworkIDState the zookeeper backed framework id state for this application
	FrameworkIDState *state.FrameworkIDState
	// Curator the zookeeper client library
	Curator *rvb_zk.Curator
	// Driver the driver for the mesos framework
	driver mesos.SchedulerDriver
}

func resourceOffer(driver *mesos.SchedulerDriver, offers []mesos.Offer) {
	log.Notice("Received %d offers.", len(offers))

	for i := range offers {
		offer := offers[i]
		if !launched {
			launched = true
			taskID, _ := flake.NewFlake().Next()
			task := mesos.TaskInfo{
				Name: proto.String("exeggutor-go-task"),
				TaskId: &mesos.TaskID{
					Value: proto.String("exeggutor-go-task-" + taskID),
				},
				SlaveId: offer.SlaveId,
				Command: &mesos.CommandInfo{
					Value: proto.String("java -jar /Users/ivan/projects/wordnik/exeggutor/sample/target/exeggutor-sample-assembly.jar"),
					Environment: &mesos.Environment{
						Variables: []*mesos.Environment_Variable{
							&mesos.Environment_Variable{
								Name:  proto.String("PORT"),
								Value: proto.String("8001"),
							},
						},
					},
				},
				Resources: []*mesos.Resource{
					mesos.ScalarResource("cpus", 1),
					mesos.ScalarResource("mem", 512),
				},
			}

			driver.LaunchTasks(offer.GetId(), []mesos.TaskInfo{task})
		} else {
			driver.DeclineOffer(offer.GetId())
		}
	}
}

// NewMesosScheduler creates a new instance of MesosScheduler with the specified config
func NewMesosScheduler(config *exeggutor.Config) *MesosScheduler {
	log.Debug("Creating a new instance of a mesos scheduler")
	return &MesosScheduler{config: config}
}

// Start initializes the scheduler and everything it depends on
func (m *MesosScheduler) Start() error {
	uri := m.config.ZookeeperUrl
	hosts, node, err := rvb_zk.ParseZookeeperUri(uri)
	if err != nil {
		log.Critical("%v", err)
		return err
	}
	conn, evt, err := zk.Connect(hosts, 5*time.Second)

	if err != nil {
		log.Critical("%v", err)
		return err
	}

	for {
		e := <-evt
		if e.State == zk.StateHasSession {
			log.Info("Connected to zookeeper(s) on %s", uri)
			break
		}
	}
	m.Curator = rvb_zk.NewCurator(conn, node)
	if err != nil {
		log.Critical("Couldn't connect to zookeeper because %v", err)
		return err
	}
	m.FrameworkIDState = state.NewFrameworkIDState(node+"/framework/id", m.Curator)
	m.FrameworkIDState.Start(true)

	master := m.config.MesosMaster
	log.Debug("Creating mesos scheduler driver")
	m.driver = mesos.SchedulerDriver{
		Master: master,
		Framework: mesos.FrameworkInfo{
			User: proto.String("ivan"),
			Name: proto.String("ExeggutorFramework"),
			Id:   m.FrameworkIDState.Get(),
		},
		Scheduler: &mesos.Scheduler{
			Registered: func(driver *mesos.SchedulerDriver, fwID mesos.FrameworkID, masterInfo mesos.MasterInfo) {
				log.Info("registered framework %v with master %v", fwID.GetValue(), masterInfo.GetId())
				m.FrameworkIDState.Set(&fwID)
			},
			OfferRescinded: func(driver *mesos.SchedulerDriver, offer mesos.OfferID) {
				log.Info("the offer %s was rescinded", offer.GetValue())
			},
			Disconnected: func(driver *mesos.SchedulerDriver) {
				log.Warning("Disconnected from master!")
			},
			Reregistered: func(driver *mesos.SchedulerDriver, masterInfo mesos.MasterInfo) {
				log.Info("Re-registered with master %s:%d\n", masterInfo.GetHostname(), masterInfo.GetPort())
			},
			SlaveLost: func(driver *mesos.SchedulerDriver, slaveID mesos.SlaveID) {
				log.Warning("Lost slave", slaveID.GetValue())
			},
			Error: func(driver *mesos.SchedulerDriver, message string) {
				log.Error("Got an error:", message)
			},
			StatusUpdate: func(driver *mesos.SchedulerDriver, status mesos.TaskStatus) {
				log.Info("Status update: %+v", status)
			},
			FrameworkMessage: func(driver *mesos.SchedulerDriver, executorID mesos.ExecutorID, slaveID mesos.SlaveID, data string) {
				log.Info("Got framework message from executor %s, slave %s, and data: %s\n", executorID.GetValue(), slaveID.GetValue(), data)
			},
			ExecutorLost: func(driver *mesos.SchedulerDriver, executorID mesos.ExecutorID, slaveID mesos.SlaveID, status int) {
				log.Error("Lost executor %s, on slave %s with status code %d\n", executorID.GetValue(), slaveID.GetValue(), status)
			},
			ResourceOffers: resourceOffer,
		},
	}
	err = m.driver.Init()
	if err != nil {
		log.Critical("Couldn't initialize the mesos scheduler driver, because %v", err)
		return err
	}
	err = m.driver.Start()
	if err != nil {
		log.Critical("Couldn't start the mesos scheduler driver, because %v", err)
		return err
	}
	log.Notice("Started the exeggutor scheduler")
	return nil
}

// Stop stops the mesos scheduler driver
func (m *MesosScheduler) Stop() error {

	err1 := m.driver.Stop(false)
	m.driver.Destroy()
	err2 := m.FrameworkIDState.Stop()
	m.Curator.Close()

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
