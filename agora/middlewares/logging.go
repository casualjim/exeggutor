package middlewares

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/op/go-logging"
	"github.com/rcrowley/go-metrics"
)

func init() {
	metrics.RegisterDebugGCStats(metrics.DefaultRegistry)
	go metrics.CaptureDebugGCStats(metrics.DefaultRegistry, 5e9)

	metrics.RegisterRuntimeMemStats(metrics.DefaultRegistry)
	go metrics.CaptureRuntimeMemStats(metrics.DefaultRegistry, 5e9)
}

// Logger is a middleware handler that logs the request as it goes in and the response as it goes out.
type Logger struct {
	// Logger is the log.Logger instance used to log messages with the Logger middleware
	Logger *logging.Logger
}

// NewLogger returns a new Logger instance
func NewLogger() *Logger {
	return &Logger{
		Logger: logging.MustGetLogger("Requests"),
	}
}

func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	l.Logger.Info("Started %s %s", r.Method, r.URL.Path)

	if !strings.HasPrefix(r.URL.Path, "/audit/metrics") {
		timer := metrics.GetOrRegisterTimer(r.URL.Path, metrics.DefaultRegistry)

		timer.Time(func() {
			next(rw, r)
		})
	} else {
		data := make(map[string]map[string]interface{})
		metrics.DefaultRegistry.Each(func(name string, i interface{}) {
			values := make(map[string]interface{})
			switch metric := i.(type) {
			case metrics.Counter:
				values["count"] = metric.Count()
			case metrics.Gauge:
				values["value"] = metric.Value()
			case metrics.GaugeFloat64:
				values["value"] = metric.Value()
			case metrics.Healthcheck:
				values["error"] = nil
				metric.Check()
				if err := metric.Error(); nil != err {
					values["error"] = metric.Error().Error()
				}
			case metrics.Histogram:
				h := metric.Snapshot()
				ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				values["count"] = h.Count()
				values["min"] = h.Min()
				values["max"] = h.Max()
				values["mean"] = h.Mean()
				values["stddev"] = h.StdDev()
				values["median"] = ps[0]
				values["p75"] = ps[1]
				values["p95"] = ps[2]
				values["p99"] = ps[3]
				values["p999"] = ps[4]
			case metrics.Meter:
				m := metric.Snapshot()
				values["count"] = m.Count()
				values["m1"] = m.Rate1()
				values["m5"] = m.Rate5()
				values["m15"] = m.Rate15()
				values["mean"] = m.RateMean()
			case metrics.Timer:
				t := metric.Snapshot()
				ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				duration := make(map[string]interface{})
				duration["unit"] = "nanoseconds"
				duration["min"] = t.Min()
				duration["max"] = t.Max()
				duration["mean"] = t.Mean()
				duration["stddev"] = t.StdDev()
				duration["median"] = ps[0]
				duration["p75"] = ps[1]
				duration["p95"] = ps[2]
				duration["p99"] = ps[3]
				duration["p999"] = ps[4]
				rate := make(map[string]interface{})
				rate["count"] = t.Count()
				rate["m1"] = t.Rate1()
				rate["m5"] = t.Rate5()
				rate["m15"] = t.Rate15()
				rate["mean"] = t.RateMean()
				values["rate"] = rate
				values["duration"] = duration

			}
			data[name] = values
			data[name] = values
		})
		enc := json.NewEncoder(rw)
		rw.Header().Set("Content-Type", "application/json;charset=utf-8")
		enc.Encode(data)
	}

	res := rw.(negroni.ResponseWriter)
	l.Logger.Info("Completed %s %s %v %s in %v", r.Method, r.URL.Path, res.Status(), http.StatusText(res.Status()), time.Since(start))

}
