package db

import (
	"fmt"

	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	GoLevelDBBackend = "goleveldb"
	CLevelDBBackend  = "cleveldb"
)

type DBWrapperWithBatch interface {
	dbm.DB
	Compact() error

	// Batch Function
	BatchDelete(key []byte)
	BatchSet(key []byte, value []byte)
	FlushBatch()
}

func LoadDB(dbBackend, name, directory string) (DBWrapperWithBatch, error) {
	switch dbBackend {
	case GoLevelDBBackend:
		return LoadGoLevelDB(name, directory)
	case CLevelDBBackend:
		return LoadCLevelDB(name, directory)
	default:
		return nil, fmt.Errorf("unknown db backend: %s", dbBackend)
	}
}
