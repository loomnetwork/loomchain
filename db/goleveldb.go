package db

import (
	"sync"

	"github.com/syndtr/goleveldb/leveldb/util"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type GoLevelDB struct {
	currentBatch dbm.Batch
	*dbm.GoLevelDB

	// Protecting against overwriting currentBatch variable with new batch while other operations
	// are still ongoing
	currentBatchMutex sync.RWMutex
}

func (g *GoLevelDB) Compact() error {
	return g.DB().CompactRange(util.Range{})
}

func (g *GoLevelDB) FlushBatch() {
	g.currentBatchMutex.Lock()
	g.currentBatch.Write()
	g.currentBatch = g.NewBatch()
	g.currentBatchMutex.Unlock()
}

func (g *GoLevelDB) BatchDelete(key []byte) {
	g.currentBatchMutex.RLock()
	g.currentBatch.Delete(key)
	g.currentBatchMutex.RUnlock()
}

func (g *GoLevelDB) BatchSet(key []byte, value []byte) {
	g.currentBatchMutex.RLock()
	g.currentBatch.Set(key, value)
	g.currentBatchMutex.RUnlock()
}

func LoadGoLevelDB(name, dir string) (*GoLevelDB, error) {
	db, err := dbm.NewGoLevelDB(name, dir)
	if err != nil {
		return nil, err
	}

	golevelDB := &GoLevelDB{GoLevelDB: db}
	golevelDB.currentBatch = golevelDB.NewBatch()

	return golevelDB, nil
}
