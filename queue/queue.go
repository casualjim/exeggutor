package queue

import (
	"io"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("exeggutor.queue")

// Queue an interface that represents a FIFO queue
// Throughout the application we can use the KVStore abstraction and get it to be
// replaced by consumers of this library with a mongo based store, redis, ...
type Queue interface {
	Len() (int, error)
	Push(item interface{}) error
	Poll() (interface{}, error)
	Peek() (interface{}, error)
	Start() error
	Stop() error
}

// Serializer allows for pluggable serialization
// for transports or stores that support it.
type Serializer interface {
	Read(reader io.ReadCloser) (interface{}, error)
	Write(writer *io.Writer, subject interface{}) error
	ReadBytes(data []byte) (interface{}, error)
	WriteBytes(target interface{}) ([]byte, error)
}

// IDGenerator an abstraction for a pluggable id generator
type IDGenerator interface {
	Next() (string, error)
}

// type JsonSerializer struct {

// }
