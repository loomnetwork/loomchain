// +build gcc

package db

import (
	"github.com/jmhodges/levigo"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type CLevelDB struct {
	*dbm.CLevelDB
}

func (c *CLevelDB) Compact() error {
	c.DB().CompactRange(levigo.Range{})
	return nil
}

func (g *CLevelDB) GetSnapshot() Snapshot {
	// TODO: Copy the GoLevelDB implementation, should be pretty much identical aside from
	//       type names.
	panic("Not implemented")
}

func LoadCLevelDB(name, dir string) (*CLevelDB, error) {
	db, err := dbm.NewCLevelDB(name, dir)
	if err != nil {
		return nil, err
	}

	return &CLevelDB{CLevelDB: db}, err
}
