package db

import (
	dbm "github.com/tendermint/tendermint/libs/db"
)

type MemDB struct {
	*dbm.MemDB
}

func (m *MemDB) Compact() error {
	return nil
}

func (m *MemDB) GetSnapshot() Snapshot {
	// TODO
	return m
}

func LoadMemDB() (*MemDB, error) {
	db := dbm.NewMemDB()

	return &MemDB{MemDB: db}, nil
}

func (m *MemDB) NewIterator(start, end []byte) dbm.Iterator {
	return m.MemDB.Iterator(start, end)
}

func (m *MemDB) Release() {
	// Noop
}
