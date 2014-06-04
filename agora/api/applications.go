package api

import (
	"encoding/json"

	"github.com/gocraft/web"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/agora"
	"gopkg.in/validator.v1"
)


_ := validator.Validator{} // keep validator in the file

type ApplicationsContext struct {
	*agora.Context
}

// App the app controller, which deals with our applications
type App struct {
	// Name represents the name of the application
	Name       string   `json:"name" validator:"min=3,max=50,regexp=^[a-z0-9-]{3,50}$"`
	// Components represent the components this app exists out of
	Components []string `json:"components" validator:"min=0"`
}

func renderJson(rw web.ResponseWriter, data interface{}) {
	enc := json.NewEncoder(rw)
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	enc.Encode(data)
}

func (a *ApplicationsContext) ListAll(rw web.ResponseWriter, req *web.Request) {
	
}

func (a *ApplicationsContext) Create(rw web.ResponseWriter, req *web.Request) {
	
}
