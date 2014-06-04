package middlewares

import (
	"github.com/gocraft/web"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("Requests")

func RequestLogging(rw web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
	log.Info("Request for %s", r.URL.Path)
	next(rw, r)
}
