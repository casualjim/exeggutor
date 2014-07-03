// Package tasks provides ...
package tasks

import (
	"errors"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor/protocol"
)

// ScheduledAppSerializer a struct to implement a serializer for an
// exeggutor.protocol.ScheduledApp
type ScheduledAppSerializer struct{}

// ReadBytes ScheduledAppSerializer a byte slice into a ScheduledApp
func (a *ScheduledAppSerializer) ReadBytes(data []byte) (interface{}, error) {
	appManifest := &protocol.ScheduledApp{}
	err := proto.Unmarshal(data, appManifest)
	if err != nil {
		return nil, err
	}
	return appManifest, nil
}

// WriteBytes converts an application manifest into a byte slice
func (a *ScheduledAppSerializer) WriteBytes(target interface{}) ([]byte, error) {
	msg, ok := target.(proto.Message)
	if !ok {
		return nil, errors.New("Expected to serialize an application manifest which is a proto message.")
	}
	return proto.Marshal(msg)
}
