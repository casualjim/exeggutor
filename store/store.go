package store

// KVData represents an entry in a Key/Value store
// It is only exposed in the ForEach method and is an internal structure otherwise
type KVData struct {
	Key   string
	Value []byte
}

// KVStore an interface that represents a Key/Value store
// Throughout the application we can use the KVStore abstraction and get it to be
// replaced by consumers of this library with a mongo based store, redis, ...
type KVStore interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Size() (int, error)
	Keys() ([]string, error)
	ForEachKey(iterator func(string)) error
	ForEachValue(iterator func([]byte)) error
	ForEach(iterator func(*KVData)) error
	Contains(key string) (bool, error)
	Start() error
	Stop() error
}
