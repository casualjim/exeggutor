package api

import (
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/state"
	"github.com/reverb/exeggutor/store"
)

const (
	// JSONContentType the mimetype for a json request
	JSONContentType = "application/json;charset=utf-8"
)

// APIContext the most generic context for this api
type APIContext struct {
	FrameworkIDState *state.FrameworkIDState
	Config           *exeggutor.Config
	AppStore         store.KVStore
}
