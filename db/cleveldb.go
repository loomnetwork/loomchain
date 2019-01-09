// +build gcc

package db

import (
	"github.com/jmhodges/levigo"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type CLevelDBWrapper struct {
	db *dbm.CLevelDB
}

func (c *CLevelDBWrapper) DB() dbm.DB {
	return c.db
}

func (c *CLevelDBWrapper) Compact() error {
	c.db.DB().CompactRange(levigo.Range{})
	return nil
}

func LoadCLevelDB(name, dir string) (*CLevelDBWrapper, error) {
	db, err := dbm.NewCLevelDB(name, dir)
	if err != nil {
		return nil, err
	}

	return &CLevelDBWrapper{db: db}, err
}
