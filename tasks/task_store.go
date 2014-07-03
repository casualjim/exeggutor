package tasks

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
)

// TaskStore A task store wraps a K/V store but
// deals with actual protocol.DeployedAppComponent types
// instead of with the raw bytes
// It's basically a KVStore with a serializer and an id generator.
type TaskStore interface {
	Get(key string) (*protocol.DeployedAppComponent, error)
	Save(value *protocol.DeployedAppComponent) error
	Delete(key string) error
	Size() (int, error)
	Keys() ([]string, error)
	ForEach(iterator func(*protocol.DeployedAppComponent)) error
	FilterToTaskIds(predicate func(*protocol.DeployedAppComponent) bool) ([]*mesos.TaskID, error)
	Find(predicate func(*protocol.DeployedAppComponent) bool) (*protocol.DeployedAppComponent, error)
	Contains(key string) (bool, error)
	Start() error
	Stop() error
}

// DefaultTaskStore the default implementation of the task store
type DefaultTaskStore struct {
	store store.KVStore
}

// NewTaskStore creates a new default instance of the task store
func NewTaskStore(config *exeggutor.Config) (TaskStore, error) {
	store, err := store.NewMdbStore(config.DataDirectory + "/tasks")
	if err != nil {
		return nil, err
	}
	return &DefaultTaskStore{
		store: store,
	}, nil
}

// Start starts this store
func (t *DefaultTaskStore) Start() error {
	return t.store.Start()
}

// Stop stops this store
func (t *DefaultTaskStore) Stop() error {
	return t.store.Stop()
}

// Get gets a deployed app component from the store by key
func (t *DefaultTaskStore) Get(key string) (*protocol.DeployedAppComponent, error) {
	data, err := t.store.Get(key)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return readBytes(data)
}

// Save saves a deployed app component with the specified id
func (t *DefaultTaskStore) Save(value *protocol.DeployedAppComponent) error {
	log.Debug("Saving %+v to the task store", value)
	ser, err := writeBytes(value)
	if err != nil {
		log.Error("Couldn't serialize deployed app component %+v, because %+v", value, err)
		return err
	}
	return t.store.Set(value.TaskId.GetValue(), ser)
}

// Delete removes the specified app component from the store
func (t *DefaultTaskStore) Delete(key string) error {
	return t.store.Delete(key)
}

// Size the amount of items stored in this store
func (t *DefaultTaskStore) Size() (int, error) {
	return t.store.Size()
}

// Keys gets all the keys in the store
func (t *DefaultTaskStore) Keys() ([]string, error) {
	return t.store.Keys()
}

// ForEach iterates over every value in the store, calling the iterator
// function for each value it sees
func (t *DefaultTaskStore) ForEach(iterator func(*protocol.DeployedAppComponent)) error {
	return t.store.ForEach(func(item *store.KVData) {
		// TODO: Use a label to get out of this loop when things go bad?
		// For now log the error and move on
		deploy, err := readBytes(item.Value)
		if err != nil {
			log.Warning("Couldn't deserialize value for %v, because %v", item.Key, err)
		}
		iterator(deploy)
	})
}

// FilterToTaskIds returns an array of task ids that match the predicate
func (t *DefaultTaskStore) FilterToTaskIds(predicate func(*protocol.DeployedAppComponent) bool) ([]*mesos.TaskID, error) {
	var result []*mesos.TaskID
	err := t.ForEach(func(item *protocol.DeployedAppComponent) {
		if predicate(item) {
			result = append(result, item.TaskId)
		}
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Find finds the first item that matches the predicate
func (t *DefaultTaskStore) Find(predicate func(*protocol.DeployedAppComponent) bool) (*protocol.DeployedAppComponent, error) {
	var lastTested *protocol.DeployedAppComponent // deserialize only once

	_, err := t.store.Find(func(kv *store.KVData) bool {
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
func (t *DefaultTaskStore) Contains(key string) (bool, error) {
	return t.store.Contains(key)
}

func readBytes(data []byte) (*protocol.DeployedAppComponent, error) {
	deploy := &protocol.DeployedAppComponent{}
	err := proto.Unmarshal(data, deploy)
	if err != nil {
		return nil, err
	}
	return deploy, nil
}

func writeBytes(target *protocol.DeployedAppComponent) ([]byte, error) {
	return proto.Marshal(target)
}
