package api

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/reverb/exeggutor/scheduler"
)

//MesosController contains the context for mesos related api calls
type MesosController struct {
}

//NewMesosController creates a new instance of mesos controller
func NewMesosController() *MesosController {
	return &MesosController{}
}

type fwID struct {
	Value *string `json:"frameworkId"`
}

//ShowFrameworkID shows the framework id of this application
func (a MesosController) ShowFrameworkID(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	state := scheduler.FrameworkIDState.Get()
	id := state.GetValue()
	enc := json.NewEncoder(rw)
	enc.Encode(&fwID{Value: &id})
}
