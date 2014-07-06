package check

import (
	"net"
	"strings"
	"time"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor/protocol"
)

var log = logging.MustGetLogger("exeggutor.health.check")

// Result the result of a health check
type Result struct {
	Code      protocol.HealthCheckResultCode
	Reason    string
	NextCheck time.Time
}

// HealthCheck is an interface that describes a strategy for health checking
// Currently supported are TCP, HTTP and METRICS, where Metrics is a specialization
// of http with a body analyzer
type HealthCheck interface {
	GetID() string
	Check() Result
	Update(config *protocol.HealthCheck)
}

func errorResult(err error, next time.Time) Result {
	log.Debug("Making error result for %v at %v", err, next)
	e, ok := err.(net.Error)
	if (ok && e.Timeout()) || strings.Contains(err.Error(), "timeout") {
		return Result{Code: protocol.HealthCheckResultCode_TIMEDOUT, NextCheck: next}
	}
	return Result{Code: protocol.HealthCheckResultCode_DOWN, NextCheck: next}
}

func successResult(next time.Time) Result {
	return Result{Code: protocol.HealthCheckResultCode_HEALTHY, NextCheck: next}
}

func faultyResult(next time.Time) Result {
	return Result{Code: protocol.HealthCheckResultCode_ERROR, NextCheck: next}
}

// New creates a new health check based on the provided configuration
func New(id, address string, config *protocol.HealthCheck) HealthCheck {
	switch config.GetMode() {

	case protocol.HealthCheckMode_HTTP:
		return newHTTPHealthCheck(id, address, config, StatusCodeValidator)
	case protocol.HealthCheckMode_METRICS:
		// TODO: reconfigure this to use the coda hale health check body for failures
		return newHTTPHealthCheck(id, address, config, StatusCodeValidator)
	default:
		return newTCPHealthCheck(id, address, config)
	}
}
