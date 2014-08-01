package model

import (
	"strings"
	"time"

	"github.com/astaxie/beego/validation"
)

// App the app controller, which deals with our applications
type App struct {
	// Name represents the name of the application
	Name string `json:"name" valid:"Required;MinSize(3);MaxSize(50);AlphaDash"`
	// Components represent the components this app exists out of
	Components map[string]AppComponent `json:"components"`
}

// Valid validates this struct
func (a App) Valid(v *validation.Validation) {
	if len(a.Components) == 0 {
		v.SetError("components", "requires at least 1 entry")
	}
}

// AppComponent a component of an application,
// in many cases there will be only one of these
// but some services require nginx etc
type AppComponent struct {

	// Name the name of the application, this is the unique identifier for an application
	Name string `json:"name" valid:"Required;MinSize(3);MaxSize(50);AlphaDash"`

	// Cpus an integer number representing a percentage of cpus it should use.
	// This is a relative scale to other services.
	Cpus int8 `json:"cpus" valid:"Min(1),Max(100)"`

	// Mem an integer number representing the number of megabytes this component needs
	// to function properly
	Mem int16 `json:"mem" valid:"Min(1)"`

	// DiskSpace an integer number representing the amount of megabytes this component
	// needs for its local working storage. This storage is transient and there
	// are no guarantees that this will be there again when an application restarts
	DiskSpace int32 `json:"disk_space,omitempty"`

	// DistUrl the url to retrieve the package from
	DistURL string `json:"dist_url" valid:"Required;MinSize(10);Match(/^\w+:\/\//)"`

	// Command the command to run for starting this component
	Command string `json:"command,omitempty"`

	// Env a map with environment variables
	Env map[string]string `json:"env"`

	// Ports a map of scheme to port
	Ports map[string]int `json:"ports"`

	// Version the version of this component
	Version string `json:"version" valid:"Required,Match(/^\d+\.\d+\.d+/)"`

	// ComponentType the type of component this is (SERVICE, TASK, CRON, SPARK_JOB)
	ComponentType string `json:"component_type"`

	// Active whether this application is available for deployment or not.
	Active bool `json:"active" valid:"Required"`

	// SLA the sla for this application if there is any
	SLA *AppSLA `json:"sla"`
}

// Valid validates this struct
func (a AppComponent) Valid(v *validation.Validation) {
	// log.Info("The app component looks: %+v", a)

	// TODO: replace this distribution thing with just a check to a docker
	//       registry and invalidate if there is no image found for an
	//       app with the calculated id from this data

	switch strings.ToUpper(a.ComponentType) {
	case "SERVICE":
		if len(a.Ports) == 0 {
			v.SetError("ports", "requires at least 1 port")
		}
	case "TASK", "CRON", "SPARK_JOB":
		v.SetError("component_type", "Only long running services are supported at the moment.")
	default:
		v.SetError("component_type", a.ComponentType+" is not supported as component type.")
	}
}

// AppSLA an application SLA describes how to check for health of a service
// as well as how many instances need to be deployed within bounds
type AppSLA struct {
	// MinInstances the minimum amount of instances that need to be deployed of this app
	MinInstances int `json:"min_instances" valid:"Required,Min(1)"`
	// MaxInstances the maximum amount of instances that can be deployed by this app
	MaxInstances int `json:"max_instances" valid:"Min(1)"`
	// HealthCheck the health check strategy to use
	HealthCheck *HealthCheck `json:"healthcheck" valid:"Required"`
}

// Valid validates an AppSLA
func (s AppSLA) Valid(v *validation.Validation) {
	if s.MaxInstances < s.MinInstances {
		v.SetError("max_instances", "Max instances needs to be larger than or equal to min instances")
	}
}

// HealthCheck describes the health check that is configured for an application
type HealthCheck struct {
	// Mode the mode for this health check (TCP, HTTP, METRICS)
	Mode string `json:"mode" valid:"Required"`
	// Rampup the rampup for this metric
	Rampup time.Duration `json:"rampup" valid:"Required"`
	// Interval the interval at which this healthcheck should occur
	Interval time.Duration `json:"interval" valid:"Required"`
	// Timeout the timeout for the healthcheck, tripping this fails the health check
	Timeout time.Duration `json:"timeout" valid:"Required"`
	// Path the path for the health check, defaults to /api/api-docs
	Path string `json:"path"`
	// Scheme the scheme for the health check, defaults to http
	Scheme string `json:"scheme"`
}

func (h HealthCheck) Valid(v *validation.Validation) {
	if h.Timeout > 1*time.Hour {
		v.SetError("timeout", "The timeout needs to be less than an hour")
	}
	if h.Timeout < 1*time.Millisecond {
		v.SetError("timeout", "It's highly unlikely that a network call will complete in less than a millisecond")
	}

	if h.Rampup > 1*time.Hour {
		v.SetError("rampup", "A rampup time of an hour is unacceptable, this service needs to be fixed")
	}
	if h.Rampup < 1*time.Millisecond {
		v.SetError("rampup", "It's highly unlikely that a network call will complete in less than a millisecond")
	}

	if h.Interval < 100*time.Millisecond {
		v.SetError("interval", "An interval should at least be 100 milliseconds, we don't want to flood the service with health checks")
	}
	if h.Interval > 5*time.Minute {
		v.SetError("interval", "An interval can at most be 5 minutes.")
	}

	if h.Mode != "TCP" && h.Mode != "HTTP" && h.Mode != "METRICS" {
		v.SetError("mode", "Mode must be one of 'tcp', 'http' or 'metrics'")
	}

}
