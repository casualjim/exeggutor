package state

import (
	"github.com/op/go-logging"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.state")

type FrameworkIDState interface {
	Path() string
	Get() *mesos.FrameworkID
	Set(fwID *mesos.FrameworkID)
	Start(buildInitial bool) error
	Stop() error
}
