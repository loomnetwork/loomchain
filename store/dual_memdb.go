package store

import (
	"github.com/tendermint/tendermint/libs/db"

	"github.com/loomnetwork/loomchain/log"
)

type dualMemDb struct {
	db.MemDB
	diskDb    db.DB
	diskBatch *difBatch
}

func newDualMemDb(diskDb db.DB) *dualMemDb {
	memDb := db.NewMemDB()

	log.Info("copy appdb into memory, this might take a while")
	for iter := diskDb.Iterator(nil, nil); iter.Valid(); iter.Next() {
		memDb.Set(iter.Key(), iter.Value())
	}
	log.Info("finished copying appdb into memroy")

	return &dualMemDb{
		MemDB:     *memDb,
		diskDb:    diskDb,
		diskBatch: NewDifBatch(diskDb),
	}
}

func (dmdb *dualMemDb) NewBatch() db.Batch {
	return &dualBatch{
		batchActive: dmdb.MemDB.NewBatch(),
		batchSilent: dmdb.diskBatch,
	}
}

func (dmdb *dualMemDb) writeToDisk() {
	dmdb.Mutex().Lock()
	defer dmdb.Mutex().Unlock()
	dmdb.diskBatch.WriteSync()
	dmdb.diskBatch.Reset()
}

type dualBatch struct {
	batchActive db.Batch
	batchSilent db.Batch
}

func (b *dualBatch) Set(key, value []byte) {
	b.batchActive.Set(key, value)
	b.batchSilent.Set(key, value)
}

func (b *dualBatch) Delete(key []byte) {
	b.batchActive.Delete(key)
	b.batchSilent.Delete(key)
}

func (b *dualBatch) Write() {
	b.batchActive.Write()
}

func (b *dualBatch) WriteSync() {
	b.batchActive.WriteSync()
}

func NewDifBatch(_db db.DB) *difBatch {
	return &difBatch{
		differences: make(map[string][]byte),
		writeDb:     _db,
	}
}

type difBatch struct {
	differences map[string][]byte
	writeDb     db.DB
}

func (b *difBatch) Set(key, value []byte) {
	b.differences[string(key)] = value
}

func (b *difBatch) Delete(key []byte) {
	b.differences[string(key)] = nil
}

func (b *difBatch) Write() {
	batch := b.writeDb.NewBatch()
	b.fillBatch(batch)
	batch.Write()
}

func (b *difBatch) WriteSync() {
	batch := b.writeDb.NewBatch()
	b.fillBatch(batch)
	batch.WriteSync()
}

func (b *difBatch) fillBatch(batch db.Batch) {
	for key, value := range b.differences {
		if value == nil {
			if b.writeDb.Has([]byte(key)) {
				batch.Delete([]byte(key))
			}
		} else {
			batch.Set([]byte(key), value)
		}
	}
}

func (b *difBatch) Reset() {
	b.differences = make(map[string][]byte)
}
