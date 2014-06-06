package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	// "github.com/reverb/exeggutor/protocol"
	"github.com/astaxie/beego/validation"
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
		log.Fatal("failed to decode:", err)
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

func unknownErrorWithMessge(rw http.ResponseWriter, err error) {
	log.Printf("There was an error: %v\n", err)
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

func validateData(rw http.ResponseWriter, data interface{}) (bool, error) {
	valid := validation.Validation{}
	b, err := valid.Valid(data)
	if err != nil {
		unknownErrorWithMessge(rw, err)
		return b, err
	}
	if !b {
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
func (a *ApplicationsController) ListAll(rw http.ResponseWriter, req *http.Request, _ map[string]string) {
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
func (a *ApplicationsController) ShowOne(rw http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	data, err := a.AppStore.Get(pathParams["name"])

	if err != nil {
		unknownErrorWithMessge(rw, err)
		return
	}

	if data == nil {
		notFound(rw, "App", pathParams["name"])
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(data)
}

// Save saves an app in the data store
func (a *ApplicationsController) Save(rw http.ResponseWriter, req *http.Request, _ map[string]string) {
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
func (a *ApplicationsController) Delete(rw http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	err := a.AppStore.Delete(pathParams["name"])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error, %v"}`, err)))
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}
