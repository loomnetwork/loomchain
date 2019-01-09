package db

import (
	"fmt"

	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	GOLevelDBBackend = "goleveldb"
	CLevelDBBackend  = "cleveldb"
)

func LoadDB(dbBackend, name, directory string, compactOnLoad bool) (dbm.DB, error, error) {
	switch dbBackend {
	case GOLevelDBBackend:
		return LoadGoLevelDB(name, directory, compactOnLoad)
	case CLevelDBBackend:
		db, err := LoadCLevelDB(name, directory, compactOnLoad)
		return db, nil, err
	default:
		return nil, nil, fmt.Errorf("unknown db backend: %s", dbBackend)
	}
}
