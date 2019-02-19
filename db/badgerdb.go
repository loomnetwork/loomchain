package db

import (
	dbm "github.com/tendermint/tendermint/libs/db"
)

type BadgerDB struct {
	*dbm.BadgerDB
}

func (b *BadgerDB) Compact() error {
	// Flatten can be used to force compactions on the LSM tree so all the tables fall on the same
	// level. This ensures that all the versions of keys are colocated and not split across multiple
	// levels, which is necessary after a restore from backup.
	return b.DB().Flatten(1)
}

func LoadBadgerDB(name, dir string) (*BadgerDB, error) {
	db, err := dbm.NewBadgerDB(name, dir)
	if err != nil {
		return nil, err
	}

	return &BadgerDB{BadgerDB: db}, nil
}
