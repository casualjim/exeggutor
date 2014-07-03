package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/scheduler"
	"github.com/reverb/exeggutor/store"
)

const (
	// JSONContentType the mimetype for a json request
	JSONContentType = "application/json;charset=utf-8"
)

// APIContext the most generic context for this api
type APIContext struct {
	Framework *scheduler.Framework
	Config    *exeggutor.Config
	AppStore  store.KVStore
}

var log = logging.MustGetLogger("agora.api")

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

func invalidJSON(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte(`{"message":"The json provided in the request is unparseable.", "type": "error"}`))
}

func unknownError(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(`{"message":"Unkown error", "type": "error"}`))
}

func unknownErrorWithMessage(rw http.ResponseWriter, err error) {
	log.Debug("There was an error: %v\n", err)
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(fmt.Sprintf(`{"message":"Unkown error: %v", "type": "error"}`, err)))
}

func notFound(rw http.ResponseWriter, name, id string) {
	rw.WriteHeader(http.StatusNotFound)
	if id == "" {
		rw.Write([]byte(fmt.Sprintf(`{"message":"Couldn't find %s.", "type": "error"}`, name)))
		return
	}
	rw.Write([]byte(fmt.Sprintf(`{"message":"Couldn't find %s for key '%s'.", "type": "error"}`, name, id)))
}
