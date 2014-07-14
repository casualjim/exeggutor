package exeggutor

import (
	"github.com/antage/eventsource"
	"github.com/robfig/cron"
)

// AppContext contains the global singleton services this application uses
// they are available in most places throughout the application
type AppContext struct {
	EventSource *eventsource.EventSource
	Cron        *cron.Cron
	Config      *Config
	IDGenerator IDGenerator
	Mesos       struct {
		Host string
		Port int
	}
}
