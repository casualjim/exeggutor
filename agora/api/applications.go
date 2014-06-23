package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"code.google.com/p/goprotobuf/proto"

	// "github.com/reverb/exeggutor/protocol"
	"github.com/astaxie/beego/validation"
	"github.com/julienschmidt/httprouter"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
)

// ApplicationsController has the context for the applications resource
// contains the applications store DAO.
type ApplicationsController struct {
	apiContext *APIContext
	AppStore   store.KVStore
}

func renderJSON(rw http.ResponseWriter, data interface{}) {
	enc := json.NewEncoder(rw)
	enc.Encode(data)
}

func readJSON(req *http.Request, data interface{}) error {
	var dec = json.NewDecoder(req.Body)
	err := dec.Decode(data)
	if err != nil {
		log.Critical("failed to decode:", err)
		return err
	}
	return nil
}

func readAppJSON(req *http.Request) (App, error) {
	var app = App{}
	err := readJSON(req, &app)
	if err != nil {
		return app, err
	}
	for k, v := range app.Components {
		v.Name = k
	}
	return app, nil
}

func invalidJSON(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte(`{"message":"The json provided in the request is unparseable."}`))
}

func unknownError(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(`{"message":"Unkown error"}`))
}

func unknownErrorWithMessge(rw http.ResponseWriter, err error) {
	log.Debug("There was an error: %v\n", err)
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error: %v"}`, err)))
}

func notFound(rw http.ResponseWriter, name, id string) {
	rw.WriteHeader(http.StatusNotFound)
	if id == "" {
		rw.Write([]byte(fmt.Sprintf(`{"message":"Couldn't find %s."}`, name)))
		return
	}
	rw.Write([]byte(fmt.Sprintf(`{"message":"Couldn't find %s for key '%s'."}`, name, id)))
}

func validateData(rw http.ResponseWriter, data App) (bool, error) {
	valid := validation.Validation{}
	b, err := valid.Valid(data)
	// data.Valid(&valid)
	log.Debug("The app %+v is valid? %t, %+v", data, valid.HasErrors(), valid)
	if err != nil {
		unknownErrorWithMessge(rw, err)
		return b, err
	}
	if valid.HasErrors() {
		rw.WriteHeader(422)
		rw.Write([]byte("["))
		isFirst := true

		for _, err := range valid.Errors {
			if !isFirst {
				rw.Write([]byte(","))
			}
			isFirst = false
			fmtStr := `{"message":"%s","field":"%s"}`
			rw.Write([]byte(fmt.Sprintf(fmtStr, err.Message, err.Field)))
		}
		return b, nil
	}
	return true, nil
}

// NewApplicationsController creates a new instance of an applications controller
func NewApplicationsController(context *APIContext) *ApplicationsController {
	return &ApplicationsController{apiContext: context, AppStore: context.AppStore}
}

// ListAll lists all the apps currently known to this application.
func (a *ApplicationsController) ListAll(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var arr [][]byte
	// Can't write in one go, need to get the error first
	err := a.AppStore.ForEachValue(func(data []byte) {
		arr = append(arr, data)
	})

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error, %v"}`, err)))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("["))
	isFirst := true
	for _, v := range arr {
		if !isFirst {
			rw.Write([]byte(","))
		}
		isFirst = false
		rw.Write(v)
	}
	rw.Write([]byte("]"))
}

// ShowOne shows a single application with all its properties
func (a *ApplicationsController) ShowOne(rw http.ResponseWriter, req *http.Request, pathParams httprouter.Params) {
	pparam := pathParams.ByName("name")
	data, err := a.AppStore.Get(pparam)

	if err != nil {
		unknownErrorWithMessge(rw, err)
		return
	}

	if data == nil {
		notFound(rw, "App", pparam)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(data)
}

// Save saves an app in the data store
func (a *ApplicationsController) Save(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	app, err := readAppJSON(req)
	if err != nil {
		invalidJSON(rw)
		return
	}

	valid, err := validateData(rw, app)
	if !valid || err != nil {
		return // rendering happened in the validateData method, we just want to get out
	}

	data, err := json.Marshal(app)
	if err != nil {
		unknownErrorWithMessge(rw, err)
		return
	}

	a.AppStore.Set(app.Name, data)

	rw.WriteHeader(http.StatusOK)
	rw.Write(data)
}

// Delete deletes a definition from this service
func (a *ApplicationsController) Delete(rw http.ResponseWriter, req *http.Request, pathParams httprouter.Params) {
	pparam := pathParams.ByName("name")
	err := a.AppStore.Delete(pparam)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error, %v"}`, err)))
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}

// Deploy takes this application and schedules it for deploy
// or for upgrade.
func (a *ApplicationsController) Deploy(rw http.ResponseWriter, req *http.Request, pathParams httprouter.Params) {
	pparam := pathParams.ByName("name")
	log.Debug("Received a request to deploy app [%s]", pparam)
	data, err := a.AppStore.Get(pparam)

	if err != nil {
		unknownErrorWithMessge(rw, err)
		return
	}
	if data == nil {
		notFound(rw, "App", pparam)
		return
	}

	app := &App{}
	err = json.Unmarshal(data, app)
	if err != nil {
		unknownErrorWithMessge(rw, err)
		return
	}
	log.Debug("Building a manifest from app %+v", app)

	var cmps []*protocol.ApplicationComponent
	for _, comp := range app.Components {

		var env []*protocol.StringKeyValue
		for k, v := range comp.Env {
			env = append(env, &protocol.StringKeyValue{
				Key:   proto.String(k),
				Value: proto.String(v),
			})
		}

		var ports []*protocol.StringIntKeyValue
		for k, v := range comp.Ports {
			ports = append(ports, &protocol.StringIntKeyValue{
				Key:   proto.String(k),
				Value: proto.Int32(int32(v)),
			})
		}

		dist := protocol.Distribution(protocol.Distribution_value[strings.ToUpper(comp.Distribution)])
		compType := protocol.ComponentType(protocol.ComponentType_value[strings.ToUpper(comp.ComponentType)])

		cmp := &protocol.ApplicationComponent{
			Name:          proto.String(comp.Name),
			Cpus:          proto.Float32(float32(comp.Cpus)),
			Mem:           proto.Float32(float32(comp.Mem)),
			DiskSpace:     proto.Int64(0),
			DistUrl:       nil,
			Command:       proto.String(comp.Command),
			Env:           env,
			Ports:         ports,
			Version:       proto.String(comp.Version),
			LogDir:        proto.String("/var/log/" + comp.Name),
			WorkDir:       proto.String("/tmp/" + comp.Name),
			ConfDir:       proto.String("/etc/" + comp.Name),
			Distribution:  &dist,
			ComponentType: &compType,
		}
		cmps = append(cmps, cmp)
	}

	appManifest := protocol.ApplicationManifest{
		Name:       proto.String(app.Name),
		Components: cmps,
	}

	a.apiContext.Framework.SubmitApp(appManifest)

	rw.WriteHeader(http.StatusAccepted)
	rw.Write([]byte("{}"))
}
