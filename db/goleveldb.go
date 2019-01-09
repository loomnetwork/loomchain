package db

import dbm "github.com/tendermint/tendermint/libs/db"
import "github.com/syndtr/goleveldb/leveldb/util"

func LoadGoLevelDB(name, dir string, compactOnLoad bool) (dbm.DB, error, error) {
	var compactionError, err error

	db, err := dbm.NewGoLevelDB(name, dir)
	if err != nil {
		return nil, compactionError, err
	}

	if compactOnLoad {
		compactionError = db.DB().CompactRange(util.Range{})
	}

	return db, compactionError, err
}
