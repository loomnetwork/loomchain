// +build gcc

package db

import (
	"sync"

	"github.com/jmhodges/levigo"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type CLevelDB struct {
	currentBatch dbm.Batch
	*dbm.CLevelDB
	// Protecting against overwriting currentBatch variable with new batch while other operations
	// are still ongoing
	currentBatchMutex sync.RWMutex
}

func (c *CLevelDB) Compact() error {
	c.DB().CompactRange(levigo.Range{})
	return nil
}

func (c *CLevelDB) FlushBatch() {
	c.currentBatchMutex.Lock()
	c.currentBatch.Write()
	c.currentBatch = c.NewBatch()
	c.currentBatchMutex.Unlock()
}

func (c *CLevelDB) BatchDelete(key []byte) {
	c.currentBatchMutex.RLock()
	c.currentBatch.Delete(key)
	c.currentBatchMutex.RUnlock()
}

func (c *CLevelDB) BatchSet(key []byte, value []byte) {
	c.currentBatchMutex.RLock()
	c.currentBatch.Set(key, value)
	c.currentBatchMutex.RUnlock()
}

func LoadCLevelDB(name, dir string) (*CLevelDB, error) {
	db, err := dbm.NewCLevelDB(name, dir)
	if err != nil {
		return nil, err
	}

	clevelDB := &CLevelDB{CLevelDB: db}
	clevelDB.currentBatch = clevelDB.NewBatch()

	return clevelDB, nil
}
