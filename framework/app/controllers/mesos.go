package controllers

import (
	"github.com/reverb/exeggutor/framework/app/scheduler"

	"github.com/revel/revel"
)

// Mesos A controller that allows insight into mesos specific information
type Mesos struct {
	*revel.Controller
}

type fwID struct {
	Value *string `json:"frameworkId"`
}

// FrameworkID returns a json structure for the framework id of this application
func (m *Mesos) FrameworkID() revel.Result {
	state := scheduler.FrameworkIDState.Get()
	if state == nil {
		return m.RenderJson(&fwID{Value: nil})
	}
	id := state.GetValue()
	if id == "" {
		return m.RenderJson(&fwID{Value: nil})
	}
	return m.RenderJson(&fwID{Value: &id})
}
