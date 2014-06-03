package controllers

import (
	"log"

	"github.com/rcrowley/go-metrics"
	"github.com/revel/revel"
)

// Audit the audit controller, allows access to diagnostic information
type Audit struct {
	*revel.Controller
}

// Metrics returns the json report for the collected metrics in this application
func (c Audit) Metrics() revel.Result {
	log.Println("Handling metrics request")
	return c.RenderJson(metrics.DefaultRegistry)
}

// InitMetrics initializes the metrics module for this application
func InitMetrics() {
	metrics.RegisterDebugGCStats(metrics.DefaultRegistry)
	go metrics.CaptureDebugGCStats(metrics.DefaultRegistry, 5e9)

	metrics.RegisterRuntimeMemStats(metrics.DefaultRegistry)
	go metrics.CaptureRuntimeMemStats(metrics.DefaultRegistry, 5e9)
}
