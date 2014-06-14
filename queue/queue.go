package queue

import "github.com/op/go-logging"

var log = logging.MustGetLogger("exeggutor.queue")

// Queue an interface that represents a FIFO queue
// Throughout the application we can use the KVStore abstraction and get it to be
// replaced by consumers of this library with a mongo based store, redis, ...
type Queue interface {
	Len() (int, error)
	Enqueue(item interface{}) error
	Dequeue() (interface{}, error)
	Peek() (interface{}, error)
	IsEmpty() (bool, error)
	Start() error
	Stop() error
}

// Serializer allows for pluggable serialization
// for transports or stores that support it.
type Serializer interface {
	ReadBytes(data []byte) (interface{}, error)
	WriteBytes(target interface{}) ([]byte, error)
}

// IDGenerator an abstraction for a pluggable id generator
type IDGenerator interface {
	Next() (string, error)
}

// type JsonSerializer struct {

// }
