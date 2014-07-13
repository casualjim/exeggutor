package protocol

import "strings"

// ID gets the id for an application
func (app Application) ID() string {
	return strings.Join([]string{app.GetAppName(), app.GetName(), app.GetVersion()}, ":")
}
