// Package tasks provides ...
package tasks

import (
	"errors"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor/protocol"
)

// ScheduledAppComponentSerializer a struct to implement a serializer for an
// exeggutor.protocol.ScheduledAppComponent
type ScheduledAppComponentSerializer struct{}

// ReadBytes ScheduledAppComponentSerializer a byte slice into a ScheduledAppComponent
func (a *ScheduledAppComponentSerializer) ReadBytes(data []byte) (interface{}, error) {
	appManifest := &protocol.ScheduledAppComponent{}
	err := proto.Unmarshal(data, appManifest)
	if err != nil {
		return nil, err
	}
	return appManifest, nil
}

// WriteBytes converts an application manifest into a byte slice
func (a *ScheduledAppComponentSerializer) WriteBytes(target interface{}) ([]byte, error) {
	msg, ok := target.(proto.Message)
	if !ok {
		return nil, errors.New("Expected to serialize an application manifest which is a proto message.")
	}
	return proto.Marshal(msg)
}
