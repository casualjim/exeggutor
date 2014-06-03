package app

import (
	"fmt"
	"strings"

	"github.com/reverb/exeggutor/framework/app/controllers"
	"github.com/reverb/exeggutor/framework/app/scheduler"

	"github.com/revel/revel"
)

func init() {
	// Filters is the default set of global filters.
	revel.Filters = []revel.Filter{
		revel.PanicFilter,             // Recover from panics and display an error page instead.
		revel.RouterFilter,            // Use the routing table to select the right Action
		revel.FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
		TimingFilter,
		JSONOrProtobufAPI,
		revel.ParamsFilter,      // Parse parameters into Controller.Params.
		revel.ValidationFilter,  // Restore kept validation errors and save new ones from cookie.
		revel.InterceptorFilter, // Run interceptors around the action.
		revel.CompressFilter,    // Compress the result.
		revel.ActionInvoker,     // Invoke the action.
	}

	// register startup functions with OnAppStart
	// ( order dependent )
	revel.OnAppStart(controllers.InitMetrics)
	revel.OnAppStart(scheduler.Start)

	// register cleanup functions with OnAppShutdown
	// ( order dependent )
	revel.OnAppShutdown(scheduler.Stop)
}

// JSONOrProtobufAPI Guards against requests that are not protobuf or json
func JSONOrProtobufAPI(c *revel.Controller, fc []revel.Filter) {
	if strings.HasPrefix(c.Request.ContentType, "application/json") ||
		strings.HasPrefix(c.Request.ContentType, "application/x-protobuf") {
		fc[0](c, fc[1:])
	} else {
		c.Response.WriteHeader(400, "application/json;charset=utf-8")
		fmt.Fprintf(c.Response.Out, "{\"error\":\"The only allowed content types are application/json and application/x-protobuf\"}")
	}
}

// TimingFilter gathers request timings and rates for this application
func TimingFilter(c *revel.Controller, fc []revel.Filter) {
	if !strings.HasPrefix(c.Request.URL.Path, "/audit/metrics") {
		timer := metrics.NewRegisteredTimer(fmt.Sprintf("controllers.%s", c.Action), metrics.DefaultRegistry)
		timer.Time(func() {
			fc[0](c, fc[1:])
		})
	} else {
		fc[0](c, fc[1:])
	}
}
