package scheduler

import (
	"time"

	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/state"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-utils/flake"
	"github.com/reverb/go-utils/rvb_zk"
	"github.com/samuel/go-zookeeper/zk"

	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.scheduler")

var (
	// FrameworkIDState the zookeeper backed framework id state for this application
	FrameworkIDState *state.FrameworkIDState
	// Curator the zookeeper client library
	Curator rvb_zk.Curator
	// Store a generic store for data (to be renamed and specialized)
	Store store.KVStore
	// Driver the driver for the mesos framework
	driver mesos.SchedulerDriver
)

var launched = true

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

// Start initializes the scheduler and everything it depends on
func Start(config exeggutor.Config) {
	uri := config.ZookeeperUrl
	hosts, node, err := rvb_zk.ParseZookeeperUri(uri)
	if err != nil {
		log.Error("%v", err)
		return
	}
	conn, evt, err := zk.Connect(hosts, 5*time.Second)

	if err != nil {
		log.Error("%v", err)
		return
	}

	for {
		e := <-evt
		if e.State == zk.StateHasSession {
			log.Info("Connected to zookeeper(s) on %s", uri)
			break
		}
	}
	Curator := rvb_zk.NewCurator(conn, node)
	if err != nil {
		log.Panicf("Couldn't connect to zookeeper because %v", err)
	}
	FrameworkIDState = state.NewFrameworkIDState(node+"/framework/id", Curator)
	FrameworkIDState.Start(true)

	master := config.MesosMaster
	log.Debug("Creating mesos scheduler driver")
	driver = mesos.SchedulerDriver{
		Master: master,
		Framework: mesos.FrameworkInfo{
			User: proto.String("ivan"),
			Name: proto.String("ExeggutorFramework"),
			Id:   FrameworkIDState.Get(),
		},
		Scheduler: &mesos.Scheduler{
			Registered: func(driver *mesos.SchedulerDriver, fwID mesos.FrameworkID, masterInfo mesos.MasterInfo) {
				log.Info("registered framework %v with master %v", fwID.GetValue(), masterInfo.GetId())
				FrameworkIDState.Set(&fwID)
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
	err = driver.Init()
	if err != nil {
		log.Panicf("Couldn't initialize the mesos scheduler driver, because %v", err)
	}
	err = driver.Start()
	if err != nil {
		log.Panicf("Couldn't start the mesos scheduler driver, because %v", err)
	}
	log.Notice("Started the exeggutor scheduler")
}

// Stop stops the mesos scheduler driver
func Stop() {

	driver.Stop(false)
	driver.Destroy()
	FrameworkIDState.Stop()
	// Curator.Close()

	log.Notice("Stopped the mesos scheduler and relevant stores")

}
