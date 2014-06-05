package middlewares

import (
	"net/http"
	"strings"

	"github.com/gocraft/web"
)

const (
	// JSONContentType the mimetype for a json request
	JSONContentType = "application/json;charset=utf-8"
)

// JSONOnlyAPI ensures that only requests with content-type application/json go through
func JSONOnlyAPI(rw web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		rw.Header().Set("Content-Type", JSONContentType)
		next(rw, r)
	} else {
		rw.Header().Set("Content-Type", JSONContentType)
		rw.WriteHeader(http.StatusUnsupportedMediaType)
		rw.Write([]byte(`{"message":"Only application/json content types are allowed to this service."}`))
	}
}
