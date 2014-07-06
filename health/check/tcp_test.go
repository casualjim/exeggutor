package check

import (
	"net"
	"testing"
	"time"

	"code.google.com/p/goprotobuf/proto"

	"github.com/reverb/exeggutor/protocol"
	// . "github.com/reverb/go-utils/convey/matchers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTCPHealthCheck(t *testing.T) {
	Convey("A TCP Health Check", t, func() {

		Convey("should return its id", func() {
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_TCP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(10),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("tcp"),
			}
			hc := newTCPHealthCheck("blah-1", "localhost:9939", config)
			So(hc.GetID(), ShouldEqual, "blah-1")
		})

		Convey("should return ok when the connection could be made in time", func() {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			So(err, ShouldBeNil)
			defer ln.Close()

			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_TCP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(60000),
				Timeout:        proto.Int64(60000),
				Scheme:         proto.String("tcp"),
			}
			hc := newTCPHealthCheck("blah-3", ln.Addr().String(), config)
			result := hc.Check()
			So(result.Code, ShouldEqual, protocol.HealthCheckResultCode_HEALTHY)
			So(result.Reason, ShouldBeEmpty)
			So(result.NextCheck, ShouldHappenAfter, time.Now())
		})

		Convey("should return timed out when the connection times out", func() {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			So(err, ShouldBeNil)
			defer ln.Close()
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_TCP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(60000),
				Timeout:        proto.Int64(-5),
				Scheme:         proto.String("tcp"),
			}
			hc := newTCPHealthCheck("blah-1", ln.Addr().String(), config)
			result := hc.Check()
			So(result.Code, ShouldEqual, protocol.HealthCheckResultCode_TIMEDOUT)
			So(result.Reason, ShouldBeEmpty)
			So(result.NextCheck, ShouldHappenAfter, time.Now())
		})

		Convey("should return down when the connection can't be established", func() {

			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_TCP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(60000),
				Scheme:         proto.String("tcp"),
			}
			hc := newTCPHealthCheck("blah-1", "localhost:9939", config)
			result := hc.Check()
			So(result.Code, ShouldEqual, protocol.HealthCheckResultCode_DOWN)
			So(result.Reason, ShouldBeEmpty)
			So(result.NextCheck, ShouldHappenAfter, time.Now())
		})

		Convey("update changes only timeout values", func() {
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_TCP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(10),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("tcp"),
			}
			hc := createTCPHealthCheck("blah-1", "localhost:9939", config)
			newConfig := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_TCP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(500),
				Timeout:        proto.Int64(1000),
				Scheme:         proto.String("mongo"),
			}
			hc2 := createTCPHealthCheck("blah-1", "localhost:9939", config)
			hc2.Update(newConfig)
			So(hc2.Address, ShouldEqual, hc.Address)
			So(hc2.ID, ShouldEqual, hc.ID)
			So(hc2.Interval, ShouldNotEqual, hc.Interval)
			So(hc2.Scheme, ShouldEqual, hc.Scheme)
			So(hc2.Timeout, ShouldNotEqual, hc.Timeout)
			So(hc2.Timeout, ShouldEqual, 1*time.Second)
			So(hc2.Interval, ShouldEqual, 500*time.Millisecond)
		})
	})
}
