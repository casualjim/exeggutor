package model

import (
	"strings"

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

	Active bool `json:"active" valid:"Required"`
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
