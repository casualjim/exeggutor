package health

import (
	"errors"
	"fmt"
	"strings"
	"sync"
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

type healthCheckPerformer struct {
	ID            int
	Trigger       chan healthCheckTrigger
	AnnounceReady chan chan healthCheckTrigger
	Closing       chan chan bool
	Results       chan check.Result
}

type healthCheckTrigger struct {
	ReplyTo chan check.Result
	Target  *activeHealthCheck
}

func newHealthCheckPerformer(id int, announceReady chan chan healthCheckTrigger) healthCheckPerformer {
	return healthCheckPerformer{
		ID:            id,
		Trigger:       make(chan healthCheckTrigger),
		AnnounceReady: announceReady,
		Closing:       make(chan chan bool),
	}
}

func (h *healthCheckPerformer) Start() {
	log.Debug("Starting health check worker %d", h.ID)
	go func() {
		var current *activeHealthCheck
		var checkDone chan check.Result

		for {
			log.Debug("About to announce worker: %d", h.ID)
			h.AnnounceReady <- h.Trigger
			log.Debug("Worker%d reporting ready", h.ID)
			select {
			case triggered := <-h.Trigger:
				log.Debug("worker%d received work: %s", h.ID, triggered.Target.HealthCheck.GetID())
				current = triggered.Target
				checkDone = triggered.ReplyTo
				go func() {
					checkDone <- triggered.Target.Start()
				}()
			case result := <-checkDone:
				log.Debug("worker%d received result for %s as %s", h.ID, result.ID, result.Code.String())
				current = nil
				h.Results <- result
				checkDone = nil
				log.Debug("worker%d forwarded result for %s", h.ID, result.ID)
			case closec := <-h.Closing:
				log.Debug("Worker %d is closing", h.ID)
				if current != nil {
					current.Stop()
					current = nil
				}
				closec <- true
				return
			}
		}
	}()
}

func (h *healthCheckPerformer) Stop() {
	closec := make(chan bool)
	h.Closing <- closec
	<-closec
}

// HealthChecker manages all the health checks for this application
// it receives a request for a health check and schedules that check.
// It can also cancel and remove a healthcheck.
// It is meant to be used by a task manager to check the services
// the task manager is supervising and notify the task manager when a
// particular health check fails
type HealthChecker struct {
	exeggutor.Module
	context          *exeggutor.AppContext
	register         map[string]*activeHealthCheck
	queue            *healthCheckQueue
	ticker           *time.Ticker
	nrOfWorkers      int
	availableWorkers chan chan healthCheckTrigger
	workers          []healthCheckPerformer
	failures         chan check.Result
}

// New creates a new instance of the health checker scheduler.
func New(context *exeggutor.AppContext) *HealthChecker {
	nrw := context.Config.FrameworkInfo.HealthCheckConcurrency
	return &HealthChecker{
		context:          context,
		register:         make(map[string]*activeHealthCheck),
		queue:            newHealthCheckQueue(),
		ticker:           time.NewTicker(1 * time.Second),
		nrOfWorkers:      nrw,
		availableWorkers: make(chan chan healthCheckTrigger, nrw),
		workers:          make([]healthCheckPerformer, nrw),
		failures:         make(chan check.Result, nrw),
	}
}

// Start starts this instance of health checker
func (h *HealthChecker) Start() error {
	for i := 0; i < h.nrOfWorkers; i++ {
		worker := newHealthCheckPerformer(i+1, h.availableWorkers)
		h.workers[i] = worker
		worker.Start()
	}
	go h.loop()
	log.Notice("Started health checker with a worker pool of %d workers", h.nrOfWorkers)
	return nil
}

func (h *HealthChecker) loop() {
	inProgress := 0
	for {
		if h.queue.Len() == 0 {
			<-h.ticker.C
			continue
		}
		if inProgress < h.nrOfWorkers {
			item := h.queue.Pop()
			if item != nil {
				inProgress++
				log.Info("We have an item to check, waiting for an available worker: %v", item)
				worker := <-h.availableWorkers
				replyTo := make(chan check.Result)
				go func() {
					log.Debug("worker acquired, sending work")
					worker <- healthCheckTrigger{ReplyTo: replyTo, Target: item}
					log.Debug("worker acquired, work sent")
					result := <-replyTo
					log.Debug("Received reply for: %v", item.HealthCheck.GetID())
					inProgress--
					log.Debug("Changing expiration from %v to %v", item.ExpiresAt, result.NextCheck)
					item.ExpiresAt = result.NextCheck
					h.queue.Push(item)
					log.Debug("healthcheck has been requeued")
					if result.Code != protocol.HealthCheckResultCode_HEALTHY {
						log.Debug("This was a healthcheck failure, forwarding....")
						h.failures <- result
					}
				}()
			} else {
				<-h.ticker.C
			}
		}
	}
}

// Stop stops this instance of health checker
func (h *HealthChecker) Stop() error {
	w := len(h.workers)
	if w > 0 {
		wg := &sync.WaitGroup{}
		wg.Add(len(h.workers))
		for _, worker := range h.workers {
			ww := &worker
			go func() {
				ww.Stop()
				wg.Done()
			}()
		}
		wg.Wait()
	}
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
