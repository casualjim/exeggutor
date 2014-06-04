package api

import (
	"encoding/json"

	"github.com/gocraft/web"
	"github.com/reverb/exeggutor/scheduler"
	"github.com/reverb/exeggutor/state"
	// "github.com/reverb/exeggutor/protocol"
	"gopkg.in/validator.v1"
)

var v = validator.Validator{} // keep validator in the file

type Context struct {
	FrameworkIDState *state.FrameworkIDState
}

type fwID struct {
	Value *string `json:"frameworkId"`
}

// FrameworkID returns a json structure for the framework id of this application
func (m *Context) ShowFrameworkID(rw web.ResponseWriter, req *web.Request) {
	state := scheduler.FrameworkIDState.Get()
	id := state.GetValue()
	enc := json.NewEncoder(rw)
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	enc.Encode(&fwID{Value: &id})
}

type ApplicationsContext struct {
	*Context
}

// App the app controller, which deals with our applications
type App struct {
	// Name represents the name of the application
	Name string `json:"name" validator:"min=3,max=50,regexp=^[a-z0-9-]{3,50}$"`
	// Components represent the components this app exists out of
	Components []string `json:"components" validator:"min=0"`
}

func renderJson(rw web.ResponseWriter, data interface{}) {
	enc := json.NewEncoder(rw)
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	enc.Encode(data)
}

func (a *ApplicationsContext) ListAll(rw web.ResponseWriter, req *web.Request) {
	rw.Write([]byte("hello"))
}

func (a *ApplicationsContext) Create(rw web.ResponseWriter, req *web.Request) {

}
