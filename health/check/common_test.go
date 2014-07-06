package check

import (
	"testing"

	"code.google.com/p/goprotobuf/proto"

	// . "github.com/reverb/go-utils/convey/matchers"

	"github.com/reverb/exeggutor/protocol"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHealthCheck(t *testing.T) {

	Convey("Creating a Healthcheck", t, func() {
		Convey("Returns a tcp health check for tcp mode", func() {
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_TCP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(10),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := New("blah1", "localhost:9939", config)
			expected := &tcpHealthCheck{}
			So(hc, ShouldHaveSameTypeAs, expected)
		})
		Convey("Returns a http health check for http mode", func() {
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(10),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := New("blah1", "localhost:9939", config)
			expected := &httpHealthCheck{}
			So(hc, ShouldHaveSameTypeAs, expected)
		})
		Convey("Returns a http health check for metrics mode", func() {
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_METRICS.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(10),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := New("blah1", "localhost:9939", config)
			expected := &httpHealthCheck{}
			So(hc, ShouldHaveSameTypeAs, expected)
		})
	})
}
