package store

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/pkg/errors"
)

type splitStore struct {
	KVReader
	VersionedKVStore
	deleted map[string]bool
}

func newSplitStore(full KVReader, empty VersionedKVStore) VersionedKVStore {
	return &splitStore{
		KVReader:         full,
		VersionedKVStore: empty,
		deleted:          make(map[string]bool),
	}
}

func (ss splitStore) Get(key []byte) []byte {
	if ss.VersionedKVStore.Has(key) {
		return ss.KVReader.Get(key)
	}
	if ss.deleted[string(key)] {
		return nil
	}
	return ss.KVReader.Get(key)
}

func (ss splitStore) Range(prefix []byte) plugin.RangeData {
	readerRange := ss.KVReader.Range(prefix)
	updateRange := ss.VersionedKVStore.Range(prefix)
	for _, re := range updateRange {
		if !ss.KVReader.Has(re.Key) && !ss.deleted[string(re.Key)] {
			readerRange = append(readerRange, re)
		}
	}
	return readerRange
}

func (ss splitStore) Has(key []byte) bool {
	if ss.VersionedKVStore.Has(key) {
		return true
	}
	if ss.deleted[string(key)] {
		return false
	}
	return ss.KVReader.Has(key)
}

func (ss splitStore) Set(key, value []byte) {
	ss.VersionedKVStore.Set(key, value)
	ss.deleted[string(key)] = false
}

func (ss splitStore) Delete(key []byte) {
	ss.VersionedKVStore.Delete(key)
	ss.deleted[string(key)] = true
}

func (ss splitStore) Hash() []byte {
	return nil
}
func (ss splitStore) Version() int64 {
	return 0
}
func (ss splitStore) SaveVersion() ([]byte, int64, error) {
	return nil, 0, nil
}

func (ss splitStore) Prune() error {
	return errors.New("not implemented")
}
func (ss splitStore) GetSnapshot(version int64) Snapshot {
	return nil
}
