package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/state"
	"github.com/reverb/exeggutor/store"
	// "github.com/reverb/exeggutor/protocol"
	"gopkg.in/validator.v1"
)

// APIContext the most generic context for this api
type APIContext struct {
	FrameworkIDState *state.FrameworkIDState
	Config           *exeggutor.Config
}

type fwID struct {
	Value *string `json:"frameworkId"`
}

// ShowFrameworkID returns a json structure for the framework id of this application
func (m *APIContext) ShowFrameworkID(rw http.ResponseWriter, req *http.Request, _ map[string]string) {

}

// ApplicationsController has the context for the applications resource
// contains the applications store DAO.
type ApplicationsController struct {
	apiContext *APIContext
	AppStore   store.KVStore
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

func renderJSON(rw http.ResponseWriter, data interface{}) {
	enc := json.NewEncoder(rw)
	enc.Encode(data)
}

func readJSON(req *http.Request, data interface{}) error {
	var dec = json.NewDecoder(req.Body)
	err := dec.Decode(data)
	if err != nil {
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

func notFound(rw http.ResponseWriter, name, id string) {
	rw.WriteHeader(http.StatusInternalServerError)
	if id == "" {
		rw.Write([]byte(fmt.Sprintf(`{"message":"Couldn't find %s."}`, name)))
		return
	}
	rw.Write([]byte(fmt.Sprintf(`{"message":"Couldn't find %s for key '%s'."}`, name, id)))
}

func validateData(rw http.ResponseWriter, data interface{}) []error {
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

// NewApplicationsController creates a new instance of an applications controller
func NewApplicationsController(context *APIContext) *ApplicationsController {
	return &ApplicationsController{apiContext: context}
}

// Start starts this module, configuring datastores etc
func (a *ApplicationsController) Start() {
	var err error
	appStore, err := store.NewMdbStore(a.apiContext.Config.DataDirectory + "/applications")
	appStore.Start()
	if err != nil {
		log.Fatalf("Couldn't initialize app database at %s/applications, because %v", a.apiContext.Config.DataDirectory, err)
	}
	a.AppStore = appStore
}

// Stop stops this module and cleans up any temp state it has
func (a *ApplicationsController) Stop() {
	if a.AppStore != nil {
		a.AppStore.Stop()
		a.AppStore = nil
	}
}

// ListAll lists all the apps currently known to this application.
func (a *ApplicationsController) ListAll(rw http.ResponseWriter, req *http.Request, _ map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
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
func (a *ApplicationsController) ShowOne(rw http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	data, err := a.AppStore.Get(pathParams["name"])

	if err != nil {
		unknownError(rw)
		return
	}

	if data == nil {
		notFound(rw, "App", pathParams["name"])
		return
	}

	rw.WriteHeader(http.StatusOK)
	renderJSON(rw, data)
}

// Save saves an app in the data store
func (a *ApplicationsController) Save(rw http.ResponseWriter, req *http.Request, _ map[string]string) {
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
func (a *ApplicationsController) Delete(rw http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	err := a.AppStore.Delete(pathParams["name"])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error, %v"}`, err)))
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}
