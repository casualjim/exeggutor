package state

import (
	"log"
	"sync"

	"github.com/reverb/go-utils/rvb_zk"

	"code.google.com/p/goprotobuf/proto"
	"github.com/reverb/go-mesos/mesos"
)

// FrameworkIDState Represents a cached value for the framework id
type FrameworkIDState struct {
	current     *mesos.FrameworkID
	cache       *rvb_zk.NodeCache
	currentLock *sync.Mutex
}

// NewFrameworkIDState creates a new instance of FrameworkIDState
func NewFrameworkIDState(path string, client *rvb_zk.Curator) *FrameworkIDState {
	return &FrameworkIDState{
		current:     nil,
		cache:       rvb_zk.NewNodeCache(path, client),
		currentLock: new(sync.Mutex),
	}
}

// Path returns the path this is referencing in zookeeper
func (f *FrameworkIDState) Path() string {
	return f.cache.Path
}

// Get gets the current framework id
func (f *FrameworkIDState) Get() *mesos.FrameworkID {
	if f.current != nil {
		return f.current
	}

	// log.Println("Finding value")
	b := f.cache.Get()
	if len(b) > 0 {
		// log.Println("We have some data")
		nw := frameworkIDFromBytes(b)
		f.Set(nw)
		return nw
	}
	return nil
}

func frameworkIDFromBytes(data []byte) *mesos.FrameworkID {
	fwID := &mesos.FrameworkID{}
	if err := proto.Unmarshal(data, fwID); err != nil {
		return nil
	}
	return fwID
}

// Set sets the framework id to a new value
func (f *FrameworkIDState) Set(fwID *mesos.FrameworkID) {
	if fwID == nil || fwID.Value == nil {
		log.Fatal("Setting FrameworkIDState to nil is not allowed")
		return
	}
	data, err := proto.Marshal(fwID)
	if err != nil {
		log.Fatalf("Unable to deserialize %+v\n%+v\n", fwID, err)
		return
	}
	f.cache.Set(data)
	f.currentLock.Lock()
	defer f.currentLock.Unlock()
	f.current = fwID
}

// Start starts the cache and optionally loads the data
func (f *FrameworkIDState) Start(buildInitial bool) error {
	return f.cache.Start(buildInitial)
}

// Stop stops the cache and clears the state
func (f *FrameworkIDState) Stop() error {
	f.currentLock.Lock()
	defer f.currentLock.Unlock()
	f.current = nil
	return f.cache.Stop()
}
