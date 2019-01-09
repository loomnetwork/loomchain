package db

import dbm "github.com/tendermint/tendermint/libs/db"
import "github.com/syndtr/goleveldb/leveldb/util"

func LoadGoLevelDB(name, dir string, compactOnLoad bool) (dbm.DB, error) {
	db, err := dbm.NewGoLevelDB(name, dir)
	if err != nil {
		return nil, err
	}

	if compactOnLoad {
		db.DB().CompactRange(util.Range{})
	}

	return db, err
}
