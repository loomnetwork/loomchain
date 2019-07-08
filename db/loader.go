package db

import (
	"fmt"

	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	GoLevelDBBackend = "goleveldb"
	CLevelDBBackend  = "cleveldb"
	MemDBackend      = "memdb"
)

type DBWrapper interface {
	dbm.DB
	Compact() error
	// GetSnapshot creates a new snapshot in a thread-safe manner.
	GetSnapshot() Snapshot
}

// Snapshot is not guaranteed to be thread-safe.
type Snapshot interface {
	Get(key []byte) []byte
	Has(key []byte) bool
	NewIterator(start, end []byte) dbm.Iterator
	Release()
}

func LoadDB(
	dbBackend, name, directory string, cacheSizeMegs int, bufferSizeMeg int, collectMetrics bool,
) (DBWrapper, error) {
	switch dbBackend {
	case GoLevelDBBackend:
		return LoadGoLevelDB(name, directory, cacheSizeMegs, bufferSizeMeg, collectMetrics)
	case CLevelDBBackend:
		return LoadCLevelDB(name, directory)
	case MemDBackend:
		return LoadMemDB()
	default:
		return nil, fmt.Errorf("unknown db backend: %s", dbBackend)
	}
}
