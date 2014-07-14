package test_utils

import (
	"github.com/reverb/exeggutor/health/sla"
	"github.com/reverb/exeggutor/protocol"
)

type NoopSLAMonitor struct {
}

func (c *NoopSLAMonitor) NeedsMoreInstances(app *protocol.Application) bool {
	return true
}
func (c *NoopSLAMonitor) CanDeployMoreInstances(app *protocol.Application) bool {
	return true
}
func (c *NoopSLAMonitor) ScaleUpOrDown() <-chan sla.ChangeDeployCount {
	return nil
}
func (c *NoopSLAMonitor) Start() error {
	return nil
}
func (c *NoopSLAMonitor) Stop() error {
	return nil
}
