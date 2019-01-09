// +build !gcc

package db

import (
	"fmt"

	dbm "github.com/tendermint/tendermint/libs/db"
)

func LoadCLevelDB(name, dir string, compactOnLoad bool) (dbm.DB, error) {
	return nil, fmt.Errorf("not enabled in non gcc build")
}
