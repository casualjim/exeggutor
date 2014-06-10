package api

import "github.com/astaxie/beego/validation"

// App the app controller, which deals with our applications
type App struct {
	// Name represents the name of the application
	Name string `json:"name" valid:"Required;MinSize(3);MaxSize(50);AlphaDash"`
	// Components represent the components this app exists out of
	Components map[string]*AppComponent `json:"components"`
}

// Valid validates this struct
func (a *App) Valid(v *validation.Validation) {
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
	Cpus int8 `json:"cpus" valid:"Range(1,100)"`
	// Mem an integer number representing the number of megabytes this component needs
	// to function properly
	Mem int16 `json:"mem" valid:"Min(1)"`
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
	// Distribution the type of distribution this component uses (PACKAGE, DOCKER, SCRIPT, FAT_JAR)
	Distribution string `json:"distribution"`
}

// Valid validates this struct
func (a *AppComponent) Valid(v *validation.Validation) {

}
