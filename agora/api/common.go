package api

import (
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
	TaskManager *scheduler.TaskManager
	Config      *exeggutor.Config
	AppStore    store.KVStore
}

var log = logging.MustGetLogger("agora.api")
