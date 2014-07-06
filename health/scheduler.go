package health

import (
	"container/heap"
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

// activeHealthCheck represents a scheduled health check
// this has a position in the health check queue based on its expiration
type activeHealthCheck struct {
	HealthCheck check.HealthCheck
	ExpiresAt   time.Time
	index       int
}

type healthCheckQueue []*activeHealthCheck

func (h healthCheckQueue) Len() int {
	return len(h)
}

func (h healthCheckQueue) Less(i, j int) bool {
	left, right := h[i], h[j]
	return left.ExpiresAt.Before(right.ExpiresAt)
}

func (h healthCheckQueue) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *healthCheckQueue) Push(x interface{}) {
	n := len(*h)
	item := x.(*activeHealthCheck)
	item.index = n
	*h = append(*h, item)
}

func (h *healthCheckQueue) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	item.index = -1
	*h = old[0 : n-1]
	return item
}

// HealthCheckScheduler spreads the work of checking several services
// concurrently but
type HealthCheckScheduler interface {
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
	queue    healthCheckQueue
}

// New creates a new instance of the health checker scheduler.
func New(context *exeggutor.AppContext) *HealthChecker {
	queue := healthCheckQueue{}
	heap.Init(&queue)
	return &HealthChecker{
		context:  context,
		register: make(map[string]*activeHealthCheck),
		queue:    queue,
	}
}

// Start starts this instance of health checker
func (h *HealthChecker) Start() error {
	return nil
}

// Stop stops this instance of health checker
func (h *HealthChecker) Stop() error {
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

func (h *HealthChecker) checkDisabled(app *protocol.DeployedAppComponent) (*protocol.HealthCheck, int32, string, string, error) {
	comp := app.GetComponent()
	id, hn := app.GetTaskId().GetValue(), app.GetHostName()
	if comp == nil {
		return nil, 0, "", "", errors.New("the component of an application can't be nil")
	}

	sla := comp.GetSla()
	if sla == nil {
		mf := "component %s for app %s has no SLA defined, disabling health check for task %s on host %s"
		log.Info(mf, app.GetAppName(), comp.GetName(), id, hn)
		return nil, 0, "", "", nil // this component doesn't need health checking
	}

	config := sla.GetHealthCheck()
	if config == nil {
		mf := "component %s for app %s has no healthcheck config, disabling health check for task %s on host %s"
		log.Info(mf, app.GetAppName(), comp.GetName(), id, hn)
		return nil, 0, "", "", nil // this component doesn't need health checking
	}

	port, ok := h.portForScheme(app.GetPortMapping(), config.GetScheme())
	if !ok {
		mf := "component %s for app %s has no ports configured, disabling health check for task %s on host %s"
		log.Info(mf, app.GetAppName(), comp.GetName(), id, hn)
		return nil, 0, "", "", nil
	}

	return config, port, id, hn, nil
}

// Register registers a health check with this component
func (h *HealthChecker) Register(app *protocol.DeployedAppComponent) error {
	config, port, id, hn, err := h.checkDisabled(app)
	if err != nil {
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
		h.register[id] = scheduled
		heap.Push(&h.queue, scheduled)
	}
	return nil
}

// Unregister unregisters and stops a health check
func (h *HealthChecker) Unregister(app *mesos.TaskID) error {
	delete(h.register, app.GetValue())
	for i, hc := range h.queue {
		if hc.HealthCheck.GetID() == app.GetValue() {
			heap.Remove(&h.queue, i)
		}
	}
	return nil
}
