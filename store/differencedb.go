package store

import (
	"github.com/tendermint/tendermint/libs/db"
)

func NewDiffIavlStore(diskDb db.DB, maxVersions, targetVersion int64, saveFrequency, versionFrequency, minCacheSize, maxCacheSize uint64) (*IAVLStore, error) {
	difDb := &differenceDb{diskDb}
	return NewIAVLStore(difDb, maxVersions, targetVersion, saveFrequency, versionFrequency, minCacheSize, maxCacheSize)
}

type differenceDb struct {
	db.DB
}

func (ddb *differenceDb) NewBatch() db.Batch {
	return &difBatch{
		differences: make(map[string][]byte),
		writeDb:     ddb.DB,
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
