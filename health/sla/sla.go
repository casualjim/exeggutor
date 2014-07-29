package sla

import (
	"time"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	app_store "github.com/reverb/exeggutor/store/apps"
	task_store "github.com/reverb/exeggutor/store/tasks"
	"github.com/reverb/exeggutor/tasks/queue"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.health.sla")

// ChangeDeployCount contains the data needed to scale an app up or down
type ChangeDeployCount struct {
	App   *protocol.Application
	Tasks []*mesos.TaskID
	Count int32
}

// SLAMonitor checks for the conditions
// that make up an SLA and allows other components
// to take action
type SLAMonitor interface {
	exeggutor.Module
	NeedsMoreInstances(app *protocol.Application) bool
	CanDeployMoreInstances(app *protocol.Application) bool
	ScaleUpOrDown() <-chan ChangeDeployCount
}

type simpleSLAMonitor struct {
	taskStore    task_store.TaskStore
	appStore     app_store.AppStore
	queue        queue.TaskQueue
	ticker       *time.Ticker
	closing      chan chan bool
	needsScaling chan ChangeDeployCount
	interval     time.Duration
}

// New creates a new instance of an SLA enforcer
func New(ts task_store.TaskStore, as app_store.AppStore, q queue.TaskQueue) SLAMonitor {
	return &simpleSLAMonitor{
		taskStore:    ts,
		appStore:     as,
		queue:        q,
		closing:      make(chan chan bool),
		needsScaling: make(chan ChangeDeployCount),
		interval:     1 * time.Minute,
	}
}

// Start starts this SLA enforcer
func (s *simpleSLAMonitor) Start() error {
	s.ticker = time.NewTicker(s.interval)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.checkSLAConformance()
			case boolc := <-s.closing:
				s.ticker.Stop()
				boolc <- true
				return
			}
		}
	}()
	return nil
}

// Stop stops this SLA enforcer
func (s *simpleSLAMonitor) Stop() error {
	boolc := make(chan bool)
	s.closing <- boolc
	<-boolc
	return nil
}

func (s *simpleSLAMonitor) shouldStopForInactive(status protocol.AppStatus) bool {
	return status != protocol.AppStatus_ABSENT &&
		status != protocol.AppStatus_FAILED &&
		status != protocol.AppStatus_STOPPED &&
		status != protocol.AppStatus_STOPPING &&
		status != protocol.AppStatus_DISABLING
}

func (s *simpleSLAMonitor) countsAsRunningForActive(status protocol.AppStatus) bool {
	return status == protocol.AppStatus_STARTED ||
		status == protocol.AppStatus_DEPLOYING ||
		status == protocol.AppStatus_UNHEALTHY
}

func (s *simpleSLAMonitor) checkSLAConformance() {
	s.appStore.ForEach(func(item *protocol.Application) {
		deployments, err := s.taskStore.Filter(func(deployment *protocol.Deployment) bool {
			return item.GetId() == deployment.GetAppId()
		})
		queuedCounts := s.queue.CountsForApps()
		if err == nil {
			if item.GetActive() {
				sla := item.GetSla()
				if sla != nil {
					// Count the apps that are actually up, being deployed or unhealthy
					// unhealthy counts as running because it uses a different lifecycle
					var count int32
					for _, deployment := range deployments {
						if s.countsAsRunningForActive(deployment.GetStatus()) {
							count++
						}
					}
					// Count the apps that are queued for deployment
					queuedCount, ok := queuedCounts[item.GetId()]
					if !ok {
						queuedCount = 0
					}
					// All the apps that count as currently deployed
					totalCount := count + queuedCount
					// if there are too few instances, scale up
					// and if there are too many instances, scale down
					if totalCount < sla.GetMinInstances() {
						s.needsScaling <- ChangeDeployCount{
							App:   item,
							Count: sla.GetMinInstances() - totalCount,
						}
					} else if totalCount > sla.GetMaxInstances() {
						s.needsScaling <- ChangeDeployCount{
							App:   item,
							Count: sla.GetMaxInstances() - totalCount, // We want a negative number here
						}
					}
				}
			} else {
				// The app is disabled, sweep up any remnants missed by buggy logic elsewhere
				var toKill []*mesos.TaskID
				for _, deployment := range deployments {
					if s.shouldStopForInactive(deployment.GetStatus()) {
						toKill = append(toKill, deployment.GetTaskId())
					}
				}
				s.needsScaling <- ChangeDeployCount{
					App:   item,
					Tasks: toKill,
					Count: int32(len(toKill) * -1),
				}
			}
		}
	})
}

func (s *simpleSLAMonitor) NeedsMoreInstances(app *protocol.Application) bool {
	runningApps := s.taskStore.RunningAppsCount(app.GetId()) + s.queue.CountAppsForID(app.GetId())
	appSLA := app.GetSla()
	if appSLA == nil {
		return false
	}
	minInstances := appSLA.GetMinInstances()
	return runningApps < minInstances
}

func (s *simpleSLAMonitor) CanDeployMoreInstances(app *protocol.Application) bool {
	appSLA := app.GetSla()
	if appSLA == nil {
		return true
	}
	runningApps := s.taskStore.RunningAppsCount(app.GetId()) + s.queue.CountAppsForID(app.GetId())
	return runningApps < appSLA.GetMaxInstances()
}

func (s *simpleSLAMonitor) ScaleUpOrDown() <-chan ChangeDeployCount {
	return s.needsScaling
}
