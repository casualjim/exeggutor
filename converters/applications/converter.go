package applications

import (
	"strings"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/agora/api/model"
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

// ToAppManifest convert the provided app to a protobuf application manifest
func (a *ApplicationsConverter) ToAppManifest(app *model.App) []protocol.Application {
	var cmps []protocol.Application
	for _, comp := range app.Components {

		var env []*protocol.StringKeyValue
		for k, v := range comp.Env {
			env = append(env, &protocol.StringKeyValue{
				Key:   proto.String(k),
				Value: proto.String(v),
			})
		}

		var ports []*protocol.StringIntKeyValue
		for k, v := range comp.Ports {
			ports = append(ports, &protocol.StringIntKeyValue{
				Key:   proto.String(k),
				Value: proto.Int32(int32(v)),
			})
		}

		dist := protocol.Distribution(protocol.Distribution_value[strings.ToUpper(comp.Distribution)])
		compType := protocol.ComponentType(protocol.ComponentType_value[strings.ToUpper(comp.ComponentType)])

		cmp := protocol.Application{
			Name:          proto.String(comp.Name),
			Cpus:          proto.Float32(float32(comp.Cpus)),
			Mem:           proto.Float32(float32(comp.Mem)),
			DiskSpace:     proto.Int64(0),
			DistUrl:       nil,
			Command:       proto.String(comp.Command),
			Env:           env,
			Ports:         ports,
			Version:       proto.String(comp.Version),
			LogDir:        nil, //proto.String("/var/log/" + comp.Name),
			WorkDir:       nil, //proto.String("/tmp/" + comp.Name),
			ConfDir:       nil, //proto.String("/etc/" + comp.Name),
			Distribution:  &dist,
			ComponentType: &compType,
			AppName:       proto.String(app.Name),
		}
		cmps = append(cmps, cmp)
	}

	return cmps
}
