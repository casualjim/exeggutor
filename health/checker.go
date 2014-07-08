package health

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/health/check"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.health")

// HealthCheckScheduler spreads the work of checking several services
// concurrently but
type HealthCheckScheduler interface {
	exeggutor.Module
	Contains(app *mesos.TaskID) bool
	Register(app *protocol.DeployedAppComponent) error
	Unregister(app *mesos.TaskID) error
	Failures() <-chan *check.Result
}

//type healthCheckPerformer struct {
//ID            int
//trigger       chan healthCheckTrigger
//announceReady chan chan healthCheckTrigger
//closing       chan chan bool
//results       chan check.Result
//}

//type healthCheckTrigger struct {
//ReplyTo chan check.Result
//Target  *activeHealthCheck
//}

//func newHealthCheckPerformer(id int, announceReady chan chan healthCheckTrigger) healthCheckPerformer {
//return healthCheckPerformer{
//ID:            id,
//trigger:       make(chan healthCheckTrigger),
//announceReady: announceReady,
//closing:       make(chan chan bool),
//}
//}

//func (h *healthCheckPerformer) Start() {
//log.Debug("Starting health check worker %d", h.ID)
//go func() {
//var current *activeHealthCheck
//var checkDone chan check.Result

//for {
//h.announceReady <- h.Trigger
//log.Debug("Worker%d reporting ready", h.ID)
//select {
//case triggered := <-h.Trigger:
//log.Debug("worker%d received work: %s", h.ID, triggered.Target.HealthCheck.GetID())
//current = triggered.Target
//checkDone = triggered.ReplyTo
//go func() {
//checkDone <- triggered.Target.Start()
//}()
//case result := <-checkDone:
//log.Debug("worker%d received result for %s as %s", h.ID, result.ID, result.Code.String())
//current = nil
//checkDone = nil
//case closec := <-h.closing:
//log.Debug("Worker %d is closing", h.ID)
//if current != nil {
//current.Stop()
//current = nil
//}
//closec <- true
//return
//}
//}
//}()
//}

//func (h *healthCheckPerformer) Stop() {
//closec := make(chan bool)
//h.closing <- closec
//<-closec
//}

func newPool(nrw int, replyTo chan<- healthResult) *workerPool {
	return &workerPool{
		poolSize: nrw,
		work:     make(chan *activeHealthCheck, nrw),
		updates:  make(chan healthResult, nrw),
		closing:  make(chan chan error),
		results:  replyTo,
	}
}

type workerPool struct {
	poolSize int
	work     chan *activeHealthCheck
	updates  chan healthResult
	closing  chan chan error
	results  chan<- healthResult
}

type healthResult struct {
	item   *activeHealthCheck
	result check.Result
}

func (p *workerPool) Start() error {
	log.Debug("Starting worker pool with %d workers", p.poolSize)
	var waitForSlot chan bool
	var pending []*activeHealthCheck
	go func() {
		for {
			select {
			case item := <-p.work:
				log.Debug("received work: %s", item.GetID())
				pending = append(pending, item)
				inProgress := len(pending)
				log.Debug("appended item to pending items, there are now %d in progress", inProgress)
				if inProgress == p.poolSize {
					log.Debug("There are as many things in progress are there are workers, setup wait for new slot")
					waitForSlot = make(chan bool)
				}
				if inProgress > p.poolSize {
					log.Warning("There are more checks in progress than there are workers!")
				}
				go func() {
					log.Debug("Perforing health check for %s", item.GetID())
					result := item.Check()
					log.Debug("healthcheck for %s finished", item.GetID())
					p.updates <- healthResult{item: item, result: result}
					if waitForSlot != nil {
						log.Debug("waitForSlot is not nil, sending it a message")
						waitForSlot <- true
					}
				}()
			case <-waitForSlot:
				log.Debug("resetting wait for slot")
				waitForSlot = nil
			case result := <-p.updates:
				log.Debug("Got a result for %s in the worker pool", result.item.GetID())
				p.results <- result
				log.Debug("pool forwarded result for %s", result.item.GetID())
				for i, item := range pending {
					if item.GetID() == result.item.GetID() {
						pending = append(pending[:i], pending[i+1:]...)
					}
				}
			case closed := <-p.closing:
				close(p.updates)
				for _, item := range pending {
					item.Cancel()
				}
				closed <- nil
			}
		}
	}()
	return nil
}

func (p *workerPool) Stop() error {
	errorc := make(chan error)
	p.closing <- errorc
	return <-errorc
}

// HealthChecker manages all the health checks for this application
// it receives a request for a health check and schedules that check.
// It can also cancel and remove a healthcheck.
// It is meant to be used by a task manager to check the services
// the task manager is supervising and notify the task manager when a
// particular health check fails
type HealthChecker struct {
	exeggutor.Module
	context  *exeggutor.AppContext
	register map[string]*activeHealthCheck
	queue    *healthCheckQueue
	ticker   *time.Ticker
	pool     *workerPool
	failures chan check.Result
	results  chan healthResult
}

// New creates a new instance of the health checker scheduler.
func New(context *exeggutor.AppContext) *HealthChecker {
	nrw := context.Config.FrameworkInfo.HealthCheckConcurrency
	results := make(chan healthResult, nrw)
	return &HealthChecker{
		context:  context,
		register: make(map[string]*activeHealthCheck),
		queue:    newHealthCheckQueue(),
		pool:     newPool(nrw, results),
		failures: make(chan check.Result),
		results:  results,
		ticker:   time.NewTicker(200 * time.Millisecond),
	}
}

// Start starts this instance of health checker
func (h *HealthChecker) Start() error {
	h.pool.Start()
	h.dequeueLoop()
	h.dispatchResultsLoop()
	log.Notice("Started health checker with a worker pool of %d workers", h.pool.poolSize)
	return nil
}

func (h *HealthChecker) dequeueLoop() {
	go func() {
		for {
			if h.queue.Len() > 0 {
				item, next, ok := h.queue.Pop()
				if ok {
					log.Debug("We have a health check to perform")
					h.pool.work <- item
				} else {
					dur := next.Sub(time.Now())
					log.Debug("No expired item found, waiting for %v", dur)
					<-time.After(dur)
				}

			} else {
				log.Debug("There were no items in the queue, waiting for a bit")
				<-h.ticker.C
			}

		}
	}()
}

func (h *HealthChecker) dispatchResultsLoop() {
	go func() {
		for result := range h.results {
			log.Debug("processing %+v", result.result)
			item := result.item
			item.ExpiresAt = result.result.NextCheck
			h.queue.Push(item)
			if result.result.Code != protocol.HealthCheckResultCode_HEALTHY {
				h.failures <- result.result
			}
		}
	}()
}

// Stop stops this instance of health checker
func (h *HealthChecker) Stop() error {
	h.pool.Stop()
	close(h.failures)

	return nil
}

func (h *HealthChecker) portForScheme(portMapping []*protocol.PortMapping, scheme string) (int32, bool) {
	for _, port := range portMapping {
		if strings.EqualFold(port.GetScheme(), scheme) {
			return port.GetPublicPort(), true
		}
	}
	return 0, false
}

func (h *HealthChecker) checkDisabled(app *protocol.DeployedAppComponent) (config *protocol.HealthCheck, port int32, id string, hn string, err error) {
	comp := app.GetComponent()
	id, hn = app.GetTaskId().GetValue(), app.GetHostName()
	if comp == nil {
		err = errors.New("the component of an application can't be nil")
		return
	}

	sla := comp.GetSla()
	if sla == nil {
		mf := "component %s for app %s has no SLA defined, disabling health check for task %s on host %s"
		log.Info(mf, app.GetAppName(), comp.GetName(), id, hn)
		return // this component doesn't need health checking
	}

	c := sla.GetHealthCheck()
	if c == nil {
		mf := "component %s for app %s has no healthcheck config, disabling health check for task %s on host %s"
		log.Info(mf, app.GetAppName(), comp.GetName(), id, hn)
		return // this component doesn't need health checking
	}

	p, ok := h.portForScheme(app.GetPortMapping(), config.GetScheme())
	if !ok {
		mf := "component %s for app %s has no ports configured, disabling health check for task %s on host %s"
		log.Info(mf, app.GetAppName(), comp.GetName(), id, hn)
		return
	}
	port, config = p, c
	return
}

// Register registers a health check with this component
func (h *HealthChecker) Register(app *protocol.DeployedAppComponent) error {
	log.Debug("Registering %+v for healthchecks", app)
	config, port, id, hn, err := h.checkDisabled(app)
	if err != nil {
		log.Error("Couldn't register app for health checks because, %v", err)
		return err
	}
	if config == nil || port == 0 {
		return nil // this was disabled
	}

	chk, ok := h.register[id]
	if ok {
		chk.HealthCheck.Update(config)
	} else {
		scheduled := &activeHealthCheck{
			HealthCheck: check.New(id, fmt.Sprintf("%s:%d", hn, port), config),
			ExpiresAt:   time.Now().Add(time.Duration(config.GetRampUp()) * time.Millisecond),
		}
		log.Debug("Enqueueing %v", scheduled)
		h.register[id] = scheduled
		h.queue.Push(scheduled)
		log.Debug("There are %d items in the queue", h.queue.Len())
	}
	return nil
}

// Unregister unregisters and stops a health check
func (h *HealthChecker) Unregister(app *mesos.TaskID) error {
	delete(h.register, app.GetValue())
	h.queue.Remove(app.GetValue())
	return nil
}

// Contains returns true when this task is known to this scheduler
func (h *HealthChecker) Contains(app *mesos.TaskID) bool {
	_, ok := h.register[app.GetValue()]
	return ok
}

func (h *HealthChecker) Failures() <-chan check.Result {
	return h.failures
}
