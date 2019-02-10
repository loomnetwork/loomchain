package db

import dbm "github.com/tendermint/tendermint/libs/db"

type MemDB struct {
	*dbm.MemDB
}

func (m *MemDB) Compact() error {
	return nil
}

func LoadMemDB() (*MemDB, error) {
	db := dbm.NewMemDB()

	return &MemDB{MemDB: db}, nil
}
