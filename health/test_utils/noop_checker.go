package test_utils

import (
	"github.com/reverb/exeggutor/health/check"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/go-mesos/mesos"
)

type NoopHealthChecker struct {
}

func (n *NoopHealthChecker) Start() error {
	return nil
}

func (n *NoopHealthChecker) Stop() error {
	return nil
}
func (n *NoopHealthChecker) Contains(app *mesos.TaskID) bool {
	return true
}
func (n *NoopHealthChecker) Register(deployment *protocol.Deployment, app *protocol.Application) error {
	return nil
}
func (n *NoopHealthChecker) Unregister(app *mesos.TaskID) error {
	return nil
}
func (n *NoopHealthChecker) Failures() <-chan check.Result {
	return nil
}
