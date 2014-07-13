package apps

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
)

var log = logging.MustGetLogger("exeggutor.apps.store")

// AppStore An app store wraps a K/V store but
// deals with actual protocol.Application types
// instead of with the raw bytes
// It's basically a KVStore with a serializer and an id generator.
type AppStore interface {
	exeggutor.Module
	Get(key string) (*protocol.Application, error)
	Save(value *protocol.Application) error
	Delete(key string) error
	Size() (int, error)
	Keys() ([]string, error)
	ForEach(iterator func(*protocol.Application)) error
	Filter(predicate func(*protocol.Application) bool) ([]*protocol.Application, error)
	Find(predicate func(*protocol.Application) bool) (*protocol.Application, error)
	Contains(key string) (bool, error)
}

// DefaultAppStore the default implementation of the app store
type DefaultAppStore struct {
	store store.KVStore
}

// New creates a new instance of the default app store
func New(config *exeggutor.Config) (AppStore, error) {
	store, err := store.NewMdbStore(config.DataDirectory + "/apps")
	if err != nil {
		return nil, err
	}
	return &DefaultAppStore{store: store}, nil
}

// NewWithStore creates a new instance of this appp store backed
// by the specified store
func NewWithStore(store store.KVStore) AppStore {
	return &DefaultAppStore{store: store}
}

// Start starts this appstore
func (a *DefaultAppStore) Start() error {
	return a.store.Start()
}

// Stop stops this app store
func (a *DefaultAppStore) Stop() error {
	return a.store.Stop()
}

// Get gets the application for that key from the store if it exists
func (a *DefaultAppStore) Get(key string) (*protocol.Application, error) {
	data, err := a.store.Get(key)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return readBytes(data)
}

// Save saves this application to the store
func (a *DefaultAppStore) Save(value *protocol.Application) error {
	log.Debug("Saving %+v to the task store", value)
	ser, err := writeBytes(value)
	if err != nil {
		log.Error("Couldn't serialize deployed app component %+v, because %+v", value, err)
		return err
	}
	return a.store.Set(value.GetId(), ser)
}

// Delete removes the specified app component from the store
func (a *DefaultAppStore) Delete(key string) error {
	return a.store.Delete(key)
}

// Size the amount of items stored in this store
func (a *DefaultAppStore) Size() (int, error) {
	return a.store.Size()
}

// Keys gets all the keys in the store
func (a *DefaultAppStore) Keys() ([]string, error) {
	return a.store.Keys()
}

// ForEach iterates over every value in the store, calling the iterator
// function for each value it sees
func (a *DefaultAppStore) ForEach(iterator func(*protocol.Application)) error {
	return a.store.ForEach(func(item *store.KVData) {
		// TODO: Use a label to get out of this loop when things go bad?
		// For now log the error and move on
		deploy, err := readBytes(item.Value)
		if err != nil {
			log.Warning("Couldn't deserialize value for %v, because %v", item.Key, err)
		}
		iterator(deploy)
	})
}

// Filter returns an array of components that match the predicate
func (a *DefaultAppStore) Filter(predicate func(*protocol.Application) bool) ([]*protocol.Application, error) {
	var result []*protocol.Application
	err := a.ForEach(func(item *protocol.Application) {
		if predicate(item) {
			result = append(result, item)
		}
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Find finds the first item that matches the predicate
func (a *DefaultAppStore) Find(predicate func(*protocol.Application) bool) (*protocol.Application, error) {
	var lastTested *protocol.Application // deserialize only once

	_, err := a.store.Find(func(kv *store.KVData) bool {
		comp, err2 := readBytes(kv.Value)
		if err2 != nil {
			log.Warning("Couldn't deserialize value for %v, because %v", kv.Key, err2)
			return false
		}
		if predicate(comp) {
			lastTested = comp
			return true
		}
		return false
	})

	if err != nil {
		return nil, err
	}
	return lastTested, nil
}

// Contains returns whether or not this store has the specified key
func (a *DefaultAppStore) Contains(key string) (bool, error) {
	return a.store.Contains(key)
}

func readBytes(data []byte) (*protocol.Application, error) {
	deploy := &protocol.Application{}
	err := proto.Unmarshal(data, deploy)
	if err != nil {
		return nil, err
	}
	return deploy, nil
}

func writeBytes(target *protocol.Application) ([]byte, error) {
	return proto.Marshal(target)
}
