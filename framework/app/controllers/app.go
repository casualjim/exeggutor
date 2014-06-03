package controllers

import (
	"github.com/reverb/exeggutor/protocol"

	"github.com/revel/revel"
)

// App the app controller, which deals with our applications
type App struct {
	*revel.Controller
}

// Index the entry point for this controller
func (c App) Index() revel.Result {
	appName := "bifrost-service"
	appStatus := protocol.AppStatus_ABSENT
	return c.RenderJson(&protocol.ApplicationManifest{
		Name:       &appName,
		Components: []*protocol.ApplicationComponent{},
		Status:     &appStatus,
	})
}
