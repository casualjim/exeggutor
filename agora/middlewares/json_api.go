package middlewares

import (
	"net/http"
	"strings"

	"github.com/reverb/exeggutor/agora/api"
)

// JSONOnlyAPI ensures that only requests with content-type application/json go through
type JSONOnlyAPI struct {
}

// NewJSONOnlyAPI creates a new instance of JSONOnlyAPI
func NewJSONOnlyAPI() *JSONOnlyAPI {
	return &JSONOnlyAPI{}
}

func (j *JSONOnlyAPI) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if !strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if strings.HasPrefix(r.URL.Path, "/api") {
			rw.Header().Set("Content-Type", api.JSONContentType)
		}
		next(rw, r)
	} else {
		rw.Header().Set("Content-Type", api.JSONContentType)
		rw.WriteHeader(http.StatusUnsupportedMediaType)
		rw.Write([]byte(`{"message":"Only application/json content types are allowed to this service."}`))
	}
}
