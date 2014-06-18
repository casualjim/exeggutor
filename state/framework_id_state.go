package state

import (
	"github.com/op/go-logging"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.state")

// FrameworkIDState is an interface that encapsulates
// keeping the framework id around in some persistence medium
type FrameworkIDState interface {
	// Path the path (key) for this interface
	Path() string
	// Get gets the framework id
	Get() *mesos.FrameworkID
	// Sets the framework id
	Set(fwID *mesos.FrameworkID)
	// Start starts the framework id state service
	Start(buildInitial bool) error
	// Stop stops the framework id state service
	Stop() error
}
