package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	// "github.com/reverb/exeggutor/protocol"
	"github.com/astaxie/beego/validation"
	"github.com/julienschmidt/httprouter"
	"github.com/reverb/exeggutor/agora/api/model"
	"github.com/reverb/exeggutor/protocol"
	app_store "github.com/reverb/exeggutor/store/apps"
)

// ApplicationsController has the context for the applications resource
// and also contains the applications store DAO.
type ApplicationsController struct {
	apiContext   *APIContext
	AppStore     app_store.AppStore
	appConverter *model.ApplicationsConverter
}

func readAppJSON(req *http.Request) (model.App, error) {
	var app = model.App{}
	err := readJSON(req, &app)
	if err != nil {
		return app, err
	}
	for k, v := range app.Components {
		v.Name = k
	}
	return app, nil
}

func validateData(rw http.ResponseWriter, data model.App) (bool, error) {
	valid := validation.Validation{}
	b, err := valid.Valid(data)
	// data.Valid(&valid)
	log.Debug("The app %+v is valid? %t, %+v", data, valid.HasErrors(), valid)
	if err != nil {
		unknownErrorWithMessage(rw, err)
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
			fmtStr := `{"message":"%s","field":"%s", "type": "error"}`
			rw.Write([]byte(fmt.Sprintf(fmtStr, err.Message, err.Field)))
		}
		return b, nil
	}
	return true, nil
}

// NewApplicationsController creates a new instance of an applications controller
func NewApplicationsController(context *APIContext) *ApplicationsController {
	return &ApplicationsController{apiContext: context, AppStore: context.AppStore, appConverter: model.New(context.Config)}
}

// ListAll lists all the apps currently known to this application.
func (a *ApplicationsController) ListAll(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var arr []*protocol.Application
	// Can't write in one go, need to get the error first
	err := a.AppStore.ForEach(func(data *protocol.Application) {
		arr = append(arr, data)
	})

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error, %v", "type": "error"}`, err)))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("["))
	isFirst := true
	enc := json.NewEncoder(rw)
	for _, v := range arr {
		if !isFirst {
			rw.Write([]byte(","))
		}
		isFirst = false
		enc.Encode(a.appConverter.FromAppManifest(v))
	}
	rw.Write([]byte("]"))
}

// ShowOne shows a single application with all its properties
func (a *ApplicationsController) ShowOne(rw http.ResponseWriter, req *http.Request, pathParams httprouter.Params) {
	pparam := pathParams.ByName("name")
	data, err := a.AppStore.Get(pparam)

	if err != nil {
		unknownErrorWithMessage(rw, err)
		return
	}

	if data == nil {
		notFound(rw, "App", pparam)
		return
	}

	rw.WriteHeader(http.StatusOK)
	d, _ := json.Marshal(a.appConverter.FromAppManifest(data))
	rw.Write(d)
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
		unknownErrorWithMessage(rw, err)
		return
	}

	for _, protoApp := range a.appConverter.ToAppManifest(&app, a.apiContext.Config) {
		a.AppStore.Save(&protoApp)
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(data)
}

// Delete deletes a definition from this service
func (a *ApplicationsController) Delete(rw http.ResponseWriter, req *http.Request, pathParams httprouter.Params) {
	pparam := pathParams.ByName("name")
	err := a.AppStore.Delete(pparam)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error, %v", "type": "error"}`, err)))
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
		unknownErrorWithMessage(rw, err)
		return
	}
	if data == nil {
		notFound(rw, "App", pparam)
		return
	}

	a.apiContext.Framework.SubmitApp([]protocol.Application{*data})

	rw.WriteHeader(http.StatusAccepted)
	d, _ := json.Marshal(a.appConverter.FromAppManifest(data))
	rw.Write(d)
}
