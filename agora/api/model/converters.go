package model

import (
	"fmt"
	"strings"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
)

// ApplicationsConverter contains the context for
// converting an exeggutor.agora.api.App to various different formats
type ApplicationsConverter struct {
	config *exeggutor.Config
}

// New create a new instance of an applications converter
func New(config *exeggutor.Config) *ApplicationsConverter {
	return &ApplicationsConverter{config: config}
}

// FromAppManifest converts an application from the backend store to the frontend representation
func (a *ApplicationsConverter) FromAppManifest(application *protocol.Application) App {
	env := make(map[string]string)
	for _, v := range application.GetEnv() {
		env[v.GetKey()] = v.GetValue()
	}

	ports := make(map[string]int)
	for _, v := range application.GetPorts() {
		ports[v.GetKey()] = int(v.GetValue())
	}

	var sla *AppSLA
	if application.Sla != nil {
		s := application.Sla
		h := s.HealthCheck

		var hc *HealthCheck
		if h != nil {
			hc = &HealthCheck{
				Mode:     h.GetMode().String(),
				Rampup:   time.Duration(h.GetRampUp()),
				Interval: time.Duration(h.GetIntervalMillis()),
				Timeout:  time.Duration(h.GetTimeout()),
				Path:     h.GetPath(),
				Scheme:   h.GetScheme(),
			}
		}

		sla = &AppSLA{
			MinInstances: int(s.GetMinInstances()),
			MaxInstances: int(s.GetMaxInstances()),
			HealthCheck:  hc,
		}
	}

	return App{
		Name: application.GetAppName(),
		Components: map[string]AppComponent{
			application.GetName(): AppComponent{
				Name:          application.GetName(),
				Cpus:          int8(application.GetCpus()),
				Mem:           int16(application.GetMem()),
				DiskSpace:     int32(application.GetDiskSpace()),
				DistURL:       application.GetDistUrl(),
				Command:       application.GetCommand(),
				Env:           env,
				Ports:         ports,
				Version:       application.GetVersion(),
				ComponentType: application.GetComponentType().String(),
				Active:        application.GetActive(),
				SLA:           sla,
			},
		},
	}
}

// ToAppManifest convert the provided app to a protobuf application manifest
func (a *ApplicationsConverter) ToAppManifest(app *App, config *exeggutor.Config) (cmps []protocol.Application) {
	for _, comp := range app.Components {

		env := []*protocol.StringKeyValue{}
		for k, v := range comp.Env {
			env = append(env, &protocol.StringKeyValue{
				Key:   proto.String(k),
				Value: proto.String(v),
			})
		}

		ports := []*protocol.StringIntKeyValue{}
		for k, v := range comp.Ports {
			ports = append(ports, &protocol.StringIntKeyValue{
				Key:   proto.String(k),
				Value: proto.Int32(int32(v)),
			})
		}

		var sla *protocol.ApplicationSLA
		if comp.SLA != nil {
			s := comp.SLA
			h := s.HealthCheck

			var hc *protocol.HealthCheck
			if h != nil {
				var mode = protocol.HealthCheckMode_HTTP
				if h.Mode != "" {
					mode = protocol.HealthCheckMode(protocol.HealthCheckMode_value[strings.ToUpper(h.Mode)])
				}
				ru := h.Rampup.Nanoseconds()

				hc = &protocol.HealthCheck{
					Mode:           mode.Enum(),
					RampUp:         proto.Int64(h.Rampup.Nanoseconds()),
					IntervalMillis: proto.Int64(h.Interval.Nanoseconds()),
					Timeout:        proto.Int64(h.Timeout.Nanoseconds()),
					Path:           proto.String(h.Path),
					Scheme:         proto.String(h.Scheme),
				}
			}
			sla = &protocol.ApplicationSLA{
				MinInstances: proto.Int32(int32(s.MinInstances)),
				MaxInstances: proto.Int32(int32(s.MaxInstances)),
				HealthCheck:  hc,
			}
		}

		appID := strings.Join([]string{app.Name, comp.Name, comp.Version}, "-")
		dist := protocol.Distribution_DOCKER.Enum()
		compType := protocol.ComponentType(protocol.ComponentType_value[strings.ToUpper(comp.ComponentType)])
		distURL := fmt.Sprintf("%s/%s/%s:%s", config.DockerIndex.ToProtoURL().String(), app.Name, comp.Name, comp.Version)

		cmp := protocol.Application{
			Id:            proto.String(appID),
			Name:          proto.String(comp.Name),
			Cpus:          proto.Float32(float32(comp.Cpus)),
			Mem:           proto.Float32(float32(comp.Mem)),
			DiskSpace:     proto.Int64(0),
			DistUrl:       proto.String(distURL),
			Command:       proto.String(comp.Command),
			Env:           env,
			Ports:         ports,
			Version:       proto.String(comp.Version),
			LogDir:        nil, //proto.String("/var/log/" + comp.Name),
			WorkDir:       nil, //proto.String("/tmp/" + comp.Name),
			ConfDir:       nil, //proto.String("/etc/" + comp.Name),
			Distribution:  dist,
			ComponentType: &compType,
			AppName:       proto.String(app.Name),
			Active:        proto.Bool(comp.Active),
			SLA:           sla,
		}
		cmps = append(cmps, cmp)
	}

	return
}
