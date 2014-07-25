package tasks

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/op/go-logging"
	"github.com/reverb/exeggutor"
	"github.com/reverb/exeggutor/protocol"
	"github.com/reverb/exeggutor/store"
	"github.com/reverb/go-mesos/mesos"
)

var log = logging.MustGetLogger("exeggutor.tasks.store")

// TaskStore A task store wraps a K/V store but
// deals with actual protocol.Deployment types
// instead of with the raw bytes
// It's basically a KVStore with a serializer and an id generator.
type TaskStore interface {
	exeggutor.Module
	Get(key string) (*protocol.Deployment, error)
	Save(value *protocol.Deployment) error
	Delete(key string) error
	Size() (int, error)
	Keys() ([]string, error)
	ForEach(iterator func(*protocol.Deployment)) error
	FilterToTaskIds(predicate func(*protocol.Deployment) bool) ([]*mesos.TaskID, error)
	Filter(predicate func(*protocol.Deployment) bool) ([]*protocol.Deployment, error)
	Find(predicate func(*protocol.Deployment) bool) (*protocol.Deployment, error)
	Contains(key string) (bool, error)
	RunningApps(appID string) ([]*protocol.Deployment, error)
	RunningAppsCount(appID string) int32
}

// DefaultTaskStore the default implementation of the task store
type DefaultTaskStore struct {
	store            store.KVStore
	runningAppsCount int32
}

// New creates a new default instance of the task store
func New(config *exeggutor.Config) (TaskStore, error) {
	store := store.NewEmptyInMemoryStore()
	return &DefaultTaskStore{store: store}, nil
}

// NewWithStore creates a new instance of a task store with a provided backing store.
// this is meant for testing, but used in another package so needed to be exported
func NewWithStore(store store.KVStore) TaskStore {
	return &DefaultTaskStore{store: store}
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
func (t *DefaultTaskStore) Get(key string) (*protocol.Deployment, error) {
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
func (t *DefaultTaskStore) Save(value *protocol.Deployment) error {
	log.Debug("Saving %+v to the task store", value)
	ser, err := writeBytes(value)
	if err != nil {
		log.Error("Couldn't serialize deployed app component %+v, because %+v", value, err)
		return err
	}

	key := value.TaskId.GetValue()
	// prev, err := t.Get(key)
	// if err != nil {
	// 	return err
	// }

	err = t.store.Set(key, ser)
	if err != nil {
		return err
	}

	// if prev != nil && prev.GetStatus() != value.GetStatus() {
	// 	if value.GetStatus() == protocol.AppStatus_STARTED {
	// 		atomic.AddInt32(&t.runningAppsCount, 1)
	// 	}
	// 	if prev.GetStatus() == protocol.AppStatus_STARTED {
	// 		atomic.AddInt32(&t.runningAppsCount, -1)
	// 	}
	// } else if prev == nil && value.GetStatus() == protocol.AppStatus_STARTED {
	// 	atomic.AddInt32(&t.runningAppsCount, 1)
	// }
	return nil
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
func (t *DefaultTaskStore) ForEach(iterator func(*protocol.Deployment)) error {
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

// Filter returns an array of components that match the predicate
func (t *DefaultTaskStore) Filter(predicate func(*protocol.Deployment) bool) ([]*protocol.Deployment, error) {
	var result []*protocol.Deployment
	err := t.ForEach(func(item *protocol.Deployment) {
		if predicate(item) {
			result = append(result, item)
		}
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// FilterToTaskIds returns an array of task ids that match the predicate
func (t *DefaultTaskStore) FilterToTaskIds(predicate func(*protocol.Deployment) bool) ([]*mesos.TaskID, error) {
	var result []*mesos.TaskID
	err := t.ForEach(func(item *protocol.Deployment) {
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
func (t *DefaultTaskStore) Find(predicate func(*protocol.Deployment) bool) (*protocol.Deployment, error) {
	var lastTested *protocol.Deployment // deserialize only once

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

// RunningApps returns the collection of deployed instances for that app
func (t *DefaultTaskStore) RunningApps(appID string) ([]*protocol.Deployment, error) {
	return t.Filter(func(i *protocol.Deployment) bool {
		return i.GetAppId() == appID && i.GetStatus() == protocol.AppStatus_STARTED
	})
}

// RunningAppsCount returns the amount of running instances for a particular application
func (t *DefaultTaskStore) RunningAppsCount(appID string) int32 {
	a, err := t.RunningApps(appID)
	if err != nil {
		return 0
	}
	return int32(len(a))
}

func readBytes(data []byte) (*protocol.Deployment, error) {
	deploy := &protocol.Deployment{}
	err := proto.Unmarshal(data, deploy)
	if err != nil {
		return nil, err
	}
	return deploy, nil
}

func writeBytes(target *protocol.Deployment) ([]byte, error) {
	return proto.Marshal(target)
}
