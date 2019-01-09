package db

import dbm "github.com/tendermint/tendermint/libs/db"
import "github.com/syndtr/goleveldb/leveldb/util"

type GoLevelDBWrapper struct {
	db *dbm.GoLevelDB
}

func (g *GoLevelDBWrapper) DB() dbm.DB {
	return g.db
}

func (g *GoLevelDBWrapper) Compact() error {
	return g.db.DB().CompactRange(util.Range{})
}

func LoadGoLevelDB(name, dir string) (*GoLevelDBWrapper, error) {
	db, err := dbm.NewGoLevelDB(name, dir)
	if err != nil {
		return nil, err
	}

	return &GoLevelDBWrapper{db: db}, nil
}
