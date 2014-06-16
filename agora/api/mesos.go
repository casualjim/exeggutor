package api

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

//MesosController contains the context for mesos related api calls
type MesosController struct {
	context *APIContext
}

//NewMesosController creates a new instance of mesos controller
func NewMesosController(context *APIContext) *MesosController {
	return &MesosController{context: context}
}

type fwID struct {
	Value string `json:"frameworkId,omitempty"`
}

//ShowFrameworkID shows the framework id of this application
func (a MesosController) ShowFrameworkID(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	id := a.context.Framework.ID()
	enc := json.NewEncoder(rw)
	enc.Encode(&fwID{Value: id})
}
