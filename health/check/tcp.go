package check

import (
	"net"
	"time"

	"github.com/reverb/exeggutor/protocol"
)

type tcpHealthCheck struct {
	ID       string
	Address  string
	Scheme   string
	Timeout  time.Duration
	Interval time.Duration
}

func createTCPHealthCheck(id, address string, config *protocol.HealthCheck) tcpHealthCheck {
	return tcpHealthCheck{
		ID:       id,
		Address:  address,
		Scheme:   config.GetScheme(),
		Timeout:  time.Duration(config.GetTimeout()) * time.Millisecond,
		Interval: time.Duration(config.GetIntervalMillis()) * time.Millisecond,
	}
}

// TCPHealthCheck creates a new health check that just checks if
// we can connect to the specified host and port
func newTCPHealthCheck(id, address string, config *protocol.HealthCheck) HealthCheck {
	hc := createTCPHealthCheck(id, address, config)
	return &hc
}

func (t *tcpHealthCheck) Check() Result {
	conn, err := net.DialTimeout("tcp", t.Address, t.Timeout)
	next := time.Now().Add(t.Interval)

	if err != nil {
		return errorResult(err, t.ID, next)
	}
	defer conn.Close()
	return successResult(t.ID, next)
}
func (t *tcpHealthCheck) GetID() string {
	return t.ID
}

// Update reconfigures a health check based on the new values
// this only reconfigures the timeout value, the interval value
func (t *tcpHealthCheck) Update(config *protocol.HealthCheck) {
	t.Timeout = time.Duration(config.GetTimeout()) * time.Millisecond
	t.Interval = time.Duration(config.GetIntervalMillis()) * time.Millisecond
}

func (t *tcpHealthCheck) Cancel() {

}
