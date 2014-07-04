package middlewares

import (
	"net/http"
	"strings"

	"github.com/antage/eventsource"
)

type EventSourceMiddleWare struct {
	eventSource eventsource.EventSource
}

func NewEventSource(es eventsource.EventSource) *EventSourceMiddleWare {
	return &EventSourceMiddleWare{eventSource: es}
}

func (e *EventSourceMiddleWare) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if strings.HasPrefix(r.URL.Path, "/api/events") {
		e.eventSource.ServeHTTP(rw, r)
	} else {
		next(rw, r)
	}
}
