// +build gcc

package db

import dbm "github.com/tendermint/tendermint/libs/db"
import "github.com/jmhodges/levigo"

func LoadCLevelDB(name, dir string, compactOnLoad bool) (dbm.DB, error) {
	db, err := dbm.NewCLevelDB(name, dir)
	if err != nil {
		return nil, err
	}

	if compactOnLoad {
		db.DB().CompactRange(levigo.Range{})
	}

	return db, err
}
