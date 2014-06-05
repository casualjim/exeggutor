package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gocraft/web"
	"github.com/reverb/exeggutor/scheduler"
	"github.com/reverb/exeggutor/state"
	"github.com/reverb/exeggutor/store"
	// "github.com/reverb/exeggutor/protocol"
	"gopkg.in/validator.v1"
)

// APIContext the most generic context for this api
type APIContext struct {
	FrameworkIDState *state.FrameworkIDState
}

type fwID struct {
	Value *string `json:"frameworkId"`
}

// ShowFrameworkID returns a json structure for the framework id of this application
func (m *APIContext) ShowFrameworkID(rw web.ResponseWriter, req *web.Request) {
	state := scheduler.FrameworkIDState.Get()
	id := state.GetValue()
	enc := json.NewEncoder(rw)
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	enc.Encode(&fwID{Value: &id})
}

// ApplicationsContext has the context for the applications resource
// contains the applications store DAO.
type ApplicationsContext struct {
	*APIContext
	AppStore store.KVStore
}

// App the app controller, which deals with our applications
type App struct {
	// Name represents the name of the application
	Name string `json:"name" validator:"min=3,max=50,regexp=^[a-z0-9-]{3,50}$"`
	// Components represent the components this app exists out of
	Components []AppComponent `json:"components" validator:"min=0"`
}

// AppComponent a component of an application,
// in many cases there will be only one of these
// but some services require nginx etc
type AppComponent struct {
	// Name the name of the application, this is the unique identifier for an application
	Name string `json:"name" validator:"min=3,max=50,regexp=^[a-z0-9-]{3,50}$"`
	// Cpus an integer number representing a percentage of cpus it should use.
	// This is a relative scale to other services.
	Cpus int8 `json:"cpus" validator:"min=1,max=100"`
	// Mem an integer number representing the number of megabytes this component needs
	// to function properly
	Mem int8 `json:"mem" validator:"min=1"`
	// DistUrl the url to retrieve the package from
	DistURL string `json:"dist_url" validator:"min=10"`
	// Command the command to run for starting this component
	Command string `json:"command,omitempty"`
	// Env a map with environment variables
	Env map[string]string `json:"env"`
	// Ports a map of scheme to port
	Ports map[string]int `json:"ports"`
	// Version the version of this component
	Version string `json:"version" validator:"regexp=^\d+\.\d+\.d+"`
	// WorkDir the working directory of this component
	WorkDir string `json:"work_dir,omitempty"`
	// Distribution the distribution type of this component (PACKAGE, DOCKER, SCRIPT, FAT_JAR)
	Distribution string `json:"distribution"`
	// ComponentType the type of component this is (SERVICE, TASK, CRON, SPARK_JOB)
	ComponentType string `json:"component_type"`
}

func renderJSON(rw web.ResponseWriter, data interface{}) {
	enc := json.NewEncoder(rw)
	enc.Encode(data)
}

func readJSON(req *web.Request, data interface{}) error {
	var dec = json.NewDecoder(req.Body)
	err := dec.Decode(data)
	if err != nil {
		return err
	}
	return nil
}

func readAppJSON(req *web.Request) (App, error) {
	var app = App{}
	err := readJSON(req, &app)
	if err != nil {
		return app, err
	}
	return app, nil
}

func invalidJSON(rw web.ResponseWriter) {
	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte(`{"message":"The json provided in the request is unparseable."}`))
}

func unknownError(rw web.ResponseWriter) {
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(`{"message":"Unkown error"}`))
}

func notFound(rw web.ResponseWriter, name, id string) {
	rw.WriteHeader(http.StatusInternalServerError)
	if id == "" {
		rw.Write([]byte(fmt.Sprintf(`{"message":"Couldn't find %s."}`, name)))
		return
	}
	rw.Write([]byte(fmt.Sprintf(`{"message":"Couldn't find %s for key '%s'."}`, name, id)))
}

func validateData(rw web.ResponseWriter, data interface{}) []error {
	if valid, errs := validator.Validate(data); !valid {
		rw.WriteHeader(http.StatusPreconditionFailed)
		rw.Write([]byte("["))

		var collected []error
		// values not valid, deal with errors here
		for k, v := range errs {
			isFirst := true
			collected = append(collected, v...)
			for _, err := range v {
				if !isFirst {
					rw.Write([]byte(","))
				}
				isFirst = false
				fmtStr := `{"message":"%s","field":"%s"}`
				rw.Write([]byte(fmt.Sprintf(fmtStr, err.Error(), k)))
			}
		}
		rw.Write([]byte("]"))
		return collected
	}
	return nil
}

// ListAll lists all the apps currently known to this application.
func (a *ApplicationsContext) ListAll(rw web.ResponseWriter, req *web.Request) {
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
	for _, v := range arr {
		isFirst := true
		if !isFirst {
			rw.Write([]byte(","))
		}
		isFirst = false
		rw.Write(v)
	}
	rw.Write([]byte("]"))
}

// ShowOne shows a single application with all its properties
func (a *ApplicationsContext) ShowOne(rw web.ResponseWriter, req *web.Request) {
	data, err := a.AppStore.Get(req.PathParams["name"])

	if err != nil {
		unknownError(rw)
		return
	}

	if data == nil {
		notFound(rw, "App", req.PathParams["name"])
		return
	}

	rw.WriteHeader(http.StatusOK)
	renderJSON(rw, data)
}

// Save saves an app in the data store
func (a *ApplicationsContext) Save(rw web.ResponseWriter, req *web.Request) {
	app, err := readAppJSON(req)
	if err != nil {
		invalidJSON(rw)
		return
	}

	errs := validateData(rw, app)
	if errs != nil {
		return // rendering happened in the validateData method, we just want to get out
	}

	data, err := json.Marshal(app)
	if err != nil {
		unknownError(rw)
		return
	}

	a.AppStore.Set(app.Name, data)

	rw.WriteHeader(http.StatusOK)
	rw.Write(data)
}

// Delete deletes a definition from this service
func (a *ApplicationsContext) Delete(rw web.ResponseWriter, req *web.Request) {
	err := a.AppStore.Delete(req.PathParams["name"])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error, %v"}`, err)))
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}
