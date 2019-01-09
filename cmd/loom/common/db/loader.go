package db

import (
	"fmt"

	dbm "github.com/tendermint/tendermint/libs/db"
)

func LoadDB(dbBackend, name, directory string, compactOnLoad bool) (dbm.DB, error) {
	switch dbBackend {
	case "leveldb":
		return LoadGoLevelDB(name, directory, compactOnLoad)
	case "cleveldb":
		return LoadCLevelDB(name, directory, compactOnLoad)
	default:
		return nil, fmt.Errorf("unknown db backend: %s", dbBackend)
	}
}
