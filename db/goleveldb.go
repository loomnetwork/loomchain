package db

import dbm "github.com/tendermint/tendermint/libs/db"
import "github.com/syndtr/goleveldb/leveldb/util"

type GoLevelDB struct {
	*dbm.GoLevelDB
}

func (g *GoLevelDB) Compact() error {
	return g.DB().CompactRange(util.Range{})
}

func LoadGoLevelDB(name, dir string) (*GoLevelDB, error) {
	db, err := dbm.NewGoLevelDB(name, dir)
	if err != nil {
		return nil, err
	}

	return &GoLevelDB{GoLevelDB: db}, nil
}
