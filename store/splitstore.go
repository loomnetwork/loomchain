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
	if ss.KVReader.Has(key) {
		return ss.KVReader.Get(key)
	}
	if ss.deleted[string(key)] {
		return nil
	}
	return ss.VersionedKVStore.Get(key)
}

func (ss splitStore) Range(prefix []byte) plugin.RangeData {
	resultRange := ss.KVReader.Range(prefix)
	diskRange := ss.VersionedKVStore.Range(prefix)
	for _, re := range diskRange {
		if !ss.KVReader.Has(re.Key) && !ss.deleted[string(re.Key)] {
			resultRange = append(resultRange, re)
		}
	}
	return resultRange
}

func (ss splitStore) Has(key []byte) bool {
	if ss.KVReader.Has(key) {
		return true
	}
	if ss.deleted[string(key)] {
		return false
	}
	return ss.VersionedKVStore.Has(key)
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
	return nil, 0, errors.New("not implemented")
}

func (ss splitStore) Prune() error {
	return errors.New("not implemented")
}
func (ss splitStore) GetSnapshot() Snapshot {
	return nil
}
