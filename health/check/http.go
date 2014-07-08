package check

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/reverb/exeggutor/protocol"
)

// ResponseValidator used to validate the health check result
// of a HTTP response
type ResponseValidator func(*http.Response, string, time.Time) Result

// StatusCodeValidator validates a http response to check if the
// response status code is in the 2xx range
func StatusCodeValidator(r *http.Response, id string, next time.Time) Result {
	if r.StatusCode >= 200 && r.StatusCode < 300 {
		return successResult(id, next)
	}
	if r.StatusCode == 504 {
		return timedOutResult(id, next)
	}
	return faultyResult(id, next)
}

type httpHealthCheck struct {
	tcpHealthCheck
	client         *http.Client
	Path           string
	validator      ResponseValidator
	currentRequest *http.Request
}

// HTTPHealthCheck creates a new health check based on
// the provided parameters
func newHTTPHealthCheck(id, address string, config *protocol.HealthCheck, validator ResponseValidator) HealthCheck {
	timeout := time.Duration(config.GetTimeout()) * time.Millisecond

	return &httpHealthCheck{
		tcpHealthCheck: createTCPHealthCheck(id, address, config),
		Path:           config.GetPath(),
		validator:      validator,
		client: &http.Client{
			Transport: &http.Transport{
				Dial:                  (&net.Dialer{Timeout: timeout}).Dial,
				DisableKeepAlives:     true,
				DisableCompression:    true,
				ResponseHeaderTimeout: timeout,
			},
		},
	}
}

func (h *httpHealthCheck) Check() Result {
	uri := fmt.Sprintf("%s://%s%s", strings.ToLower(h.Scheme), h.Address, h.Path)
	req, err := http.NewRequest("GET", uri, nil)
	next := time.Now().Add(h.Interval)
	if err != nil {
		return errorResult(err, h.ID, next)
	}
	h.currentRequest = req
	r, err := h.client.Do(h.currentRequest)
	h.currentRequest = nil

	if err != nil {
		return errorResult(err, h.ID, next)
	}
	return h.validator(r, h.ID, next)
}

// Update reconfigures a health check based on the new values
// this only reconfigures the timeout value, the interval value and the response validation
func (h *httpHealthCheck) Update(config *protocol.HealthCheck) {
	timeout := time.Duration(config.GetTimeout()) * time.Millisecond
	if timeout != h.Timeout {
		h.Timeout = time.Duration(config.GetTimeout()) * time.Millisecond
		h.client = &http.Client{
			Transport: &http.Transport{
				Dial:                  (&net.Dialer{Timeout: timeout}).Dial,
				DisableKeepAlives:     true,
				DisableCompression:    true,
				ResponseHeaderTimeout: timeout,
			},
		}
	}
	h.Interval = time.Duration(config.GetIntervalMillis()) * time.Millisecond

	if config.GetMode() == protocol.HealthCheckMode_METRICS {
		// TODO: reconfigure this to use the coda hale health check body for failures
		h.validator = StatusCodeValidator
	} else {
		h.validator = StatusCodeValidator
	}
}

func (h *httpHealthCheck) Cancel() {
	if h.currentRequest != nil {
		trans := h.client.Transport.(*http.Transport)
		trans.CancelRequest(h.currentRequest)
	}
}
