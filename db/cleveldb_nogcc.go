// +build !gcc

package db

import (
	"fmt"

	dbm "github.com/tendermint/tendermint/libs/db"
)

func LoadCLevelDB(name, dir string, compactOnLoad bool) (dbm.DB, error) {
	return nil, fmt.Errorf("DBBackend: %s is not available in build without gcc tag", CLevelDBBackend)
}
