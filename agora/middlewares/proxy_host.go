package middlewares

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// ProxyHostMiddleware holds the state for this middleware
type ProxyHostMiddleware struct {
	BasePath string
	proxy    *httputil.ReverseProxy
}

// NewProxyHost creates a new instance of a reverse proxy
func NewProxyHost(basePath string, remoteHost *url.URL) *ProxyHostMiddleware {
	return &ProxyHostMiddleware{BasePath: basePath, proxy: httputil.NewSingleHostReverseProxy(remoteHost)}
}

func (p *ProxyHostMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if strings.HasPrefix(r.URL.Path, p.BasePath) {
		remotePath := strings.TrimPrefix(r.URL.Path, p.BasePath)
		toProxy := new(http.Request)
		*toProxy = *r
		toProxy.URL.Path = remotePath
		p.proxy.ServeHTTP(rw, toProxy)
	} else {
		next(rw, r)
	}
}
