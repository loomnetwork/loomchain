package store

import (
	"github.com/loomnetwork/go-loom/plugin"
)

type splitStore struct {
	KVReader
	VersionedKVStore
	deleted map[string]bool
	version int64
}

func NewSplitStore(full KVReader, empty VersionedKVStore, version int64) VersionedKVStore {
	return &splitStore{
		KVReader:         full,
		VersionedKVStore: empty,
		deleted:          make(map[string]bool),
		version:          version,
	}
}

func (ss *splitStore) Get(key []byte) []byte {
	if ss.VersionedKVStore.Has(key) {
		return ss.VersionedKVStore.Get(key)
	}
	if ss.deleted[string(key)] {
		return nil
	}
	return ss.KVReader.Get(key)
}

func (ss *splitStore) Range(prefix []byte) plugin.RangeData {
	readerRange := ss.KVReader.Range(prefix)
	updateRange := ss.VersionedKVStore.Range(prefix)
	for _, re := range readerRange {
		if !ss.VersionedKVStore.Has(re.Key) && !ss.deleted[string(re.Key)] {
			updateRange = append(updateRange, re)
		}
	}
	return updateRange
}

func (ss *splitStore) Has(key []byte) bool {
	if ss.VersionedKVStore.Has(key) {
		return true
	}
	if ss.deleted[string(key)] {
		return false
	}
	return ss.KVReader.Has(key)
}

func (ss *splitStore) Set(key, value []byte) {
	ss.VersionedKVStore.Set(key, value)
	ss.deleted[string(key)] = false
}

func (ss *splitStore) Delete(key []byte) {
	ss.VersionedKVStore.Delete(key)
	ss.deleted[string(key)] = true
}

func (ss *splitStore) Hash() []byte {
	return []byte{}
}

func (ss *splitStore) Version() int64 {
	return ss.version
}

func (ss *splitStore) SaveVersion() ([]byte, int64, error) {
	ss.version++
	return ss.Hash(), ss.version, nil
}

func (ss *splitStore) Prune() error {
	return nil
}

func (ss *splitStore) GetSnapshot() Snapshot {
	return &splitStoreSnapShot{*ss}
}

func (ss *splitStore) GetSnapshotAt(version int64) (Snapshot, error) {
	panic("should not be called")
}

type splitStoreSnapShot struct {
	splitStore
}

func (ss *splitStoreSnapShot) Release() {}
