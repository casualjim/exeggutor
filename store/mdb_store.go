package store

import (
	"fmt"
	"log"
	"os"

	"github.com/armon/gomdb"
)

const (
	dbMaxMapSize = 128 * 1024 * 1024 // 128 MB default max map size
	defaultSize  = 0
)

// MdbStore represents a store backed by LMDB
type MdbStore struct {
	env     *mdb.Env
	path    string
	maxSize uint64
}

// NewMdbStore returns a new MDBStore and potential
// error. Requres a base directory from which to operate.
// Uses the default maximum size.
func NewMdbStore(base string) (KVStore, error) {
	return NewMdbStoreWithSize(base, defaultSize)
}

// NewMdbStoreWithSize returns a new MDBStore and potential
// error. Requres a base directory from which to operate,
// and a maximum size. If maxSize is not 0, a default value is used.
func NewMdbStoreWithSize(path string, maxSize uint64) (KVStore, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	env, err := mdb.NewEnv()
	if err != nil {
		return nil, err
	}

	store := &MdbStore{
		env:     env,
		path:    path,
		maxSize: maxSize,
	}
	return store, nil
}

// Get retrieve the []byte value from the store for the specified key
func (i MdbStore) Get(key string) ([]byte, error) {
	tx, dbis, err := i.startTxn(true)
	if err != nil {
		return nil, err
	}
	defer tx.Abort()

	val, err := tx.Get(dbis[0], []byte(key))
	if err == mdb.NotFound {
		return nil, fmt.Errorf("not found")
	} else if err != nil {
		return nil, err
	}
	return val, nil
}

// Set sets the specified to the specified value in the store
func (i MdbStore) Set(key string, value []byte) error {
	tx, dbis, err := i.startTxn(false)
	if err != nil {
		return err
	}
	if err := tx.Put(dbis[0], []byte(key), value, 0); err != nil {
		tx.Abort()
		return err
	}
	return tx.Commit()
}

// Delete deletes the specified key from the storage
func (i MdbStore) Delete(key string) error {
	tx, dbis, err := i.startTxn(false)
	if err != nil {
		return err
	}
	if err := tx.Del(dbis[0], []byte(key), nil); err != nil {
		tx.Abort()
		return err
	}
	return tx.Commit()
}

// Size gets the size for all the items contained in this store
func (i MdbStore) Size() (int, error) {
	var count int
	err := i.ForEach(func(kv *KVData) { count++ })
	return count, err
}

// Keys retrieves all the keys currently in the store
func (i MdbStore) Keys() ([]string, error) {
	var keys []string

	err := i.ForEachKey(func(key string) {
		keys = append(keys, key)
	})

	if err != nil {
		return nil, err
	}
	return keys, nil
}

// ForEachKey invokes the specified function for each key in the store
func (i MdbStore) ForEachKey(iterator func(string)) error {
	return i.ForEach(func(kv *KVData) {
		iterator(kv.Key)
	})
}

// ForEachValue invokes the specified function for each value in the store
func (i MdbStore) ForEachValue(iterator func([]byte)) error {
	return i.ForEach(func(kv *KVData) {
		iterator(kv.Value)
	})
}

// ForEach invokes the specified function for each Key/Value pair in the store
func (i MdbStore) ForEach(iterator func(*KVData)) error {
	tx, dbis, err := i.startTxn(true)
	if err != nil {
		return err
	}
	defer tx.Abort()
	cursor, err := tx.CursorOpen(dbis[0])
	if err != nil {
		return err
	}

	for {
		key, value, err := cursor.Get(nil, mdb.NEXT)
		if err == mdb.NotFound {
			break
		} else if err != nil {
			return err
		}
		iterator(&KVData{Key: string(key), Value: value})
	}
	if err := cursor.Close(); err != nil {
		log.Println("Couldn't close cursor because:", err)
	}
	return nil
}

// Contains returns true if the key exists in the store
func (i MdbStore) Contains(key string) (bool, error) {
	tx, dbis, err := i.startTxn(true)
	if err != nil {
		return false, err
	}
	defer tx.Abort()

	val, err := tx.Get(dbis[0], []byte(key))
	if err == mdb.NotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return val != nil && len(val) > 0, nil
}

// Start starts this store
func (i MdbStore) Start() error {

	// Increase the maximum map size
	if err := i.env.SetMapSize(i.maxSize); err != nil {
		return err
	}

	// Open the DB
	if err := i.env.Open(i.path, mdb.NOTLS, 0755); err != nil {
		return err
	}

	// Create all the tables
	tx, _, err := i.startTxn(false)
	if err != nil {
		tx.Abort()
		return err
	}
	return tx.Commit()

}

// Stop is used to gracefully shutdown the MDB store
func (i MdbStore) Stop() error {
	i.env.Close()
	return nil
}

// startTxn is used to start a transaction and open all the associated sub-databases
func (i MdbStore) startTxn(readonly bool) (*mdb.Txn, []mdb.DBI, error) {
	var txFlags uint
	var dbFlags uint
	if readonly {
		txFlags |= mdb.RDONLY
	} else {
		dbFlags |= mdb.CREATE
	}

	tx, err := i.env.BeginTxn(nil, txFlags)
	if err != nil {
		return nil, nil, err
	}

	var dbs []mdb.DBI
	dbi, err := tx.DBIOpen("", dbFlags)
	if err != nil {
		tx.Abort()
		return nil, nil, err
	}
	dbs = append(dbs, dbi)

	return tx, dbs, nil
}
