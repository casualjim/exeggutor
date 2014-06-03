package scheduler

import (
	"time"

	"github.com/reverb/exeggutor/state"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-util/rvb_zk"
	"github.com/reverb/go-utils/flake"

	"code.google.com/p/goprotobuf/proto"
	"github.com/revel/revel"
	"github.com/reverb/go-mesos/mesos"
	"github.com/samuel/go-zookeeper/zk"
)

var (
	// FrameworkIDState the zookeeper backed framework id state for this application
	FrameworkIDState *state.FrameworkIDState
	// Curator the zookeeper client library
	Curator *rvb_zk.Curator
	// Store a generic store for data (to be renamed and specialized)
	Store store.KVStore
	// Driver the driver for the mesos framework
	driver mesos.SchedulerDriver
)

var launched = false

func resourceOffer(driver *mesos.SchedulerDriver, offers []mesos.Offer) {
	revel.INFO.Println("Received", len(offers), "offers.")

	// executor :=

	for i := range offers {
		offer := offers[i]
		if !launched {
			launched = true
			taskId, _ := flake.NewFlake().Next()
			task := mesos.TaskInfo{
				Name: proto.String("exeggutor-go-task"),
				TaskId: &mesos.TaskID{
					Value: proto.String("exeggutor-go-task-" + taskId),
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
func Start() {
	revel.TRACE.Println("Connecting to zookeeper on localhost:2181")
	servers, recvTimeout := []string{"localhost:2181"}, 5*time.Second

	zkClient, evt, err := zk.Connect(servers, recvTimeout)
	if err != nil {
		revel.ERROR.Println(err)
	}

	for {
		e := <-evt
		if e.State == zk.StateHasSession {
			revel.INFO.Println("Connected to zookeeper on localhost:2181")
			break
		}
	}

	Curator = rvb_zk.NewCurator(zkClient)
	FrameworkIDState = state.NewFrameworkIDState("/exeggutor/framework/id", Curator)
	FrameworkIDState.Start(true)

	Store = store.NewEmptyInMemoryStore()
	Store.Start()

	master := "zk://localhost:2181/mesos"
	revel.TRACE.Println("Creating mesos scheduler driver")
	driver = mesos.SchedulerDriver{
		Master: master,
		Framework: mesos.FrameworkInfo{
			User: proto.String("ivan"),
			Name: proto.String("ExeggutorFramework"),
			Id:   FrameworkIDState.Get(),
		},
		Scheduler: &mesos.Scheduler{
			Registered: func(driver *mesos.SchedulerDriver, fwID mesos.FrameworkID, masterInfo mesos.MasterInfo) {
				revel.INFO.Println("registered framework", fwID.GetValue(), "with master", masterInfo.GetId())
				FrameworkIDState.Set(&fwID)
			},
			OfferRescinded: func(driver *mesos.SchedulerDriver, offer mesos.OfferID) {
				revel.INFO.Println("the offer", offer.GetValue(), "was rescinded")
			},
			Disconnected: func(driver *mesos.SchedulerDriver) {
				revel.WARN.Println("Disconnected from master!")
			},
			Reregistered: func(driver *mesos.SchedulerDriver, masterInfo mesos.MasterInfo) {
				revel.INFO.Printf("Re-registered with master %s:%d\n", masterInfo.GetHostname(), masterInfo.GetPort())
			},
			SlaveLost: func(driver *mesos.SchedulerDriver, slaveID mesos.SlaveID) {
				revel.WARN.Println("Lost slave", slaveID.GetValue())
			},
			Error: func(driver *mesos.SchedulerDriver, message string) {
				revel.ERROR.Println("Got an error:", message)
			},
			StatusUpdate: func(driver *mesos.SchedulerDriver, status mesos.TaskStatus) {
				revel.INFO.Printf("Status update: %+v", status)
			},
			FrameworkMessage: func(driver *mesos.SchedulerDriver, executorID mesos.ExecutorID, slaveID mesos.SlaveID, data string) {
				revel.INFO.Printf("Got framework message from executor %s, slave %s, and data: %s\n", executorID.GetValue(), slaveID.GetValue(), data)
			},
			ExecutorLost: func(driver *mesos.SchedulerDriver, executorID mesos.ExecutorID, slaveID mesos.SlaveID, status int) {
				revel.ERROR.Printf("Lost executor %s, on slave %s with status code %d\n", executorID.GetValue(), slaveID.GetValue(), status)
			},
			ResourceOffers: resourceOffer,
		},
	}
	err = driver.Init()
	if err != nil {
		revel.ERROR.Println("Couldn't initialize the mesos scheduler driver, because", err)
		panic(err)
	}
	err = driver.Start()
	if err != nil {
		revel.ERROR.Println("Couldn't start the mesos scheduler driver, because", err)
		panic(err)
	}

}

// Stop stops the mesos scheduler driver
func Stop() {
	driver.Destroy()
	driver.Stop(false)
	Store.Stop()
	FrameworkIDState.Stop()
	Curator.Close()
	revel.INFO.Println("Stopped the mesos scheduler and relevant stores")
}
