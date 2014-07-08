package check

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"code.google.com/p/goprotobuf/proto"

	// . "github.com/reverb/go-utils/convey/matchers"
	"github.com/reverb/exeggutor/protocol"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHTTPHealthCheck(t *testing.T) {

	Convey("A HTTP health check", t, func() {
		Convey("returns its id", func() {
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(1000),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := newHTTPHealthCheck("blah-1", "localhost:9939", config, StatusCodeValidator)
			So(hc.GetID(), ShouldEqual, "blah-1")
		})

		Convey("should return ok when the connection could be made in time", func() {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			So(err, ShouldBeNil)
			go http.Serve(ln, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusOK)
			}))
			defer ln.Close()

			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(10000),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := newHTTPHealthCheck("blah-1", ln.Addr().String(), config, StatusCodeValidator)
			result := hc.Check()
			So(result.Code, ShouldEqual, protocol.HealthCheckResultCode_HEALTHY)
			So(result.Reason, ShouldBeEmpty)
			So(result.NextCheck, ShouldHappenAfter, time.Now())
		})

		Convey("should return timed out when the connection times out", func() {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			So(err, ShouldBeNil)
			go http.Serve(ln, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				fmt.Println("Got a request for:", r.URL.Path)
				time.Sleep(1 * time.Second)
			}))
			defer ln.Close()

			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(10000),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := newHTTPHealthCheck("blah-1", ln.Addr().String(), config, StatusCodeValidator)
			result := hc.Check()
			So(result.Code, ShouldEqual, protocol.HealthCheckResultCode_TIMEDOUT)
			So(result.Reason, ShouldBeEmpty)
			So(result.NextCheck, ShouldHappenAfter, time.Now())
		})

		Convey("should return timed out when the status code is 504", func() {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			So(err, ShouldBeNil)
			go http.Serve(ln, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				fmt.Println("Got a request for:", r.URL.Path)
				rw.WriteHeader(http.StatusGatewayTimeout)
			}))
			defer ln.Close()

			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(1000),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := newHTTPHealthCheck("blah-1", ln.Addr().String(), config, StatusCodeValidator)
			result := hc.Check()
			So(result.Code, ShouldEqual, protocol.HealthCheckResultCode_TIMEDOUT)
			So(result.Reason, ShouldBeEmpty)
			So(result.NextCheck, ShouldHappenAfter, time.Now())
		})

		Convey("should return down when the connection can't be established", func() {
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(100),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := newHTTPHealthCheck("blah-1", "localhost:9939", config, StatusCodeValidator)
			result := hc.Check()
			So(result.Code, ShouldEqual, protocol.HealthCheckResultCode_DOWN)
			So(result.Reason, ShouldBeEmpty)
			So(result.NextCheck, ShouldHappenAfter, time.Now())
		})

		Convey("should return faulty when the status code is not in the 2xx range", func() {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			So(err, ShouldBeNil)
			go http.Serve(ln, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			}))
			defer ln.Close()

			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(1000),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("http"),
			}
			hc := newHTTPHealthCheck("blah-1", ln.Addr().String(), config, StatusCodeValidator)
			result := hc.Check()
			So(result.Code, ShouldEqual, protocol.HealthCheckResultCode_ERROR)
			So(result.Reason, ShouldBeEmpty)
			So(result.NextCheck, ShouldHappenAfter, time.Now())
		})

		Convey("update changes only timeout values", func() {
			config := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(10),
				Timeout:        proto.Int64(100),
				Scheme:         proto.String("tcp"),
			}
			hc := newHTTPHealthCheck("blah-1", "localhost:9939", config, StatusCodeValidator).(*httpHealthCheck)
			newConfig := &protocol.HealthCheck{
				Mode:           protocol.HealthCheckMode_HTTP.Enum(),
				RampUp:         proto.Int64(10),
				IntervalMillis: proto.Int64(500),
				Timeout:        proto.Int64(1000),
				Scheme:         proto.String("mongo"),
			}
			hc2 := newHTTPHealthCheck("blah-1", "localhost:9939", config, StatusCodeValidator).(*httpHealthCheck)
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
