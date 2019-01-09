package db

import (
	"fmt"

	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	GOLevelDBBackend = "goleveldb"
	CLevelDBBackend  = "cleveldb"
)

type DBWrapper interface {
	Compact() error
	DB() dbm.DB
}

func LoadDB(dbBackend, name, directory string, compactOnLoad bool) (DBWrapper, error) {
	switch dbBackend {
	case GOLevelDBBackend:
		return LoadGoLevelDB(name, directory)
	case CLevelDBBackend:
		return LoadCLevelDB(name, directory)
	default:
		return nil, fmt.Errorf("unknown db backend: %s", dbBackend)
	}
}
