package queue

import (
	"os"

	"github.com/armon/gomdb"
)

const (
	dbMaxMapSize = 128 * 1024 * 1024 // 128 MB default max map size
	defaultSize  = 0
)

// MdbQueue represents a persistent FIFO queue
// the queue is backed by a K/V store that preserves
// natural insertion order so we can actually
// iterate over things one by one
type MdbQueue struct {
	env         *mdb.Env
	path        string
	maxSize     uint64
	serializer  Serializer
	idGenerator IDGenerator
}

// NewMdbQueue returns a new MDBStore and potential
// error. Requres a base directory from which to operate.
// Uses the default maximum size.
func NewMdbQueue(base string) (*MdbQueue, error) {
	return NewMdbQueueWithSize(base, defaultSize)
}

// NewMdbQueueWithSize returns a new MDBStore and potential
// error. Requres a base directory from which to operate,
// and a maximum size. If maxSize is not 0, a default value is used.
func NewMdbQueueWithSize(path string, maxSize uint64) (*MdbQueue, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	env, err := mdb.NewEnv()
	if err != nil {
		return nil, err
	}

	store := &MdbQueue{
		env:     env,
		path:    path,
		maxSize: maxSize,
	}
	return store, nil
}

// Peek returns the next item to be dequeued if any, but does not dequeue
func (q MdbQueue) Peek() (interface{}, error) {
	tx, dbis, err := q.startTxn(true)
	if err != nil {
		return nil, err
	}
	defer tx.Abort()

	cursor, err := tx.CursorOpen(dbis[0])
	if err != nil {
		return nil, err
	}
	_, value, err := cursor.Get(nil, mdb.NEXT)
	if err == mdb.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res, err := q.serializer.ReadBytes(value)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Poll dequeues an item from this queue
func (q MdbQueue) Poll() (interface{}, error) {
	tx, dbis, err := q.startTxn(true)
	if err != nil {
		if tx != nil {
			tx.Abort()
		}
		return nil, err
	}

	cursor, err := tx.CursorOpen(dbis[0])
	if err != nil {
		tx.Abort()
		return nil, err
	}
	key, value, err := cursor.Get(nil, mdb.NEXT)
	if err == mdb.NotFound {
		tx.Abort()
		return nil, nil
	}
	if err != nil {
		tx.Abort()
		return nil, err
	}
	res, err := q.serializer.ReadBytes(value)
	if err != nil {
		tx.Abort()
		return nil, err
	}
	if err := tx.Del(dbis[0], []byte(key), nil); err != nil {
		tx.Abort()
		return nil, err
	}
	return res, tx.Commit()
}

// Push enqueues an item on this persistent queue
func (q MdbQueue) Push(item interface{}) error {
	tx, dbis, err := q.startTxn(false)
	if err != nil {
		return err
	}

	key, err := q.idGenerator.Next()
	if err != nil {
		tx.Abort()
		return err
	}

	value, err := q.serializer.WriteBytes(item)
	if err != nil {
		tx.Abort()
		return err
	}
	if err := tx.Put(dbis[0], []byte(key), value, 0); err != nil {
		tx.Abort()
		return err
	}
	return tx.Commit()
}

// Len gets the length of the queue, this operation is O(n) with the size of the queue
func (q MdbQueue) Len() (int, error) {
	var count int
	tx, dbis, err := q.startTxn(true)
	if err != nil {
		return 0, err
	}
	defer tx.Abort()

	cursor, err := tx.CursorOpen(dbis[0])
	if err != nil {
		return 0, err
	}

	for {
		_, _, err := cursor.Get(nil, mdb.NEXT)
		if err == mdb.NotFound {
			break
		}
		if err != nil {
			return count, err
		}
		count++
	}
	if err := cursor.Close(); err != nil {
		log.Warning("Couldn't close cursor because:", err)
	}
	return count, nil
}

// Start starts this queue
func (q MdbQueue) Start() error {

	// Increase the maximum map size
	if err := q.env.SetMapSize(q.maxSize); err != nil {
		return err
	}

	// Open the DB
	if err := q.env.Open(q.path, mdb.NOTLS, 0755); err != nil {
		return err
	}

	// Create all the tables
	tx, _, err := q.startTxn(false)
	if err != nil {
		tx.Abort()
		return err
	}
	return tx.Commit()

}

// Stop is used to gracefully shutdown the MDB queue
func (q MdbQueue) Stop() error {
	q.env.Close()
	return nil
}

// startTxn is used to start a transaction and open all the associated sub-databases
func (q MdbQueue) startTxn(readonly bool) (*mdb.Txn, []mdb.DBI, error) {
	var txFlags uint
	var dbFlags uint
	if readonly {
		txFlags |= mdb.RDONLY
	} else {
		dbFlags |= mdb.CREATE
	}

	tx, err := q.env.BeginTxn(nil, txFlags)
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
