package store

// InMemoryStore represents a data store that is in memory and thus transient
type InMemoryStore struct {
	data map[string][]byte
}

// NewInMemoryStore create a new instance of a memory store with seed data
func NewInMemoryStore(initialData map[string][]byte) KVStore {
	return &InMemoryStore{data: initialData}
}

// NewEmptyInMemoryStore creates a new instance of an in memory store without seed data
func NewEmptyInMemoryStore() KVStore {
	return &InMemoryStore{data: make(map[string][]byte)}
}

// Get retrieve the []byte value from the store for the specified key
func (i InMemoryStore) Get(key string) ([]byte, error) {
	return i.data[key], nil
}

// Set sets the specified to the specified value in the store
func (i InMemoryStore) Set(key string, value []byte) error {
	i.data[key] = value
	return nil
}

// Delete deletes the specified key from the storage
func (i InMemoryStore) Delete(key string) error {
	delete(i.data, key)
	return nil
}

// Size gets the size for all the items contained in this store
func (i InMemoryStore) Size() (int, error) {
	return len(i.data), nil
}

// Keys retrieves all the keys currently in the store
func (i InMemoryStore) Keys() ([]string, error) {
	var keys []string
	for k := range i.data {
		if k != "" {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// ForEachKey invokes the specified function for each key in the store
func (i InMemoryStore) ForEachKey(iterator func(string)) error {
	for k := range i.data {
		iterator(k)
	}
	return nil
}

// ForEachValue invokes the specified function for each value in the store
func (i InMemoryStore) ForEachValue(iterator func([]byte)) error {
	for _, v := range i.data {
		iterator(v)
	}
	return nil
}

// ForEach invokes the specified function for each Key/Value pair in the store
func (i InMemoryStore) ForEach(iterator func(*KVData)) error {
	for k, v := range i.data {
		iterator(&KVData{Key: k, Value: v})
	}
	return nil
}

// Find finds the first item in the store that matches the predicate
func (i InMemoryStore) Find(predicate func(*KVData) bool) (*KVData, error) {
	for k, v := range i.data {
		kv := &KVData{Key: k, Value: v}
		if predicate(kv) {
			return kv, nil
		}
	}
	return nil, nil
}

// Contains returns true if the key exists in the store
func (i InMemoryStore) Contains(key string) (bool, error) {
	return len(i.data[key]) > 0, nil
}

// Start starts this store
func (i InMemoryStore) Start() error {
	return nil
}

// Stop stops this store
func (i InMemoryStore) Stop() error {
	return nil
}
