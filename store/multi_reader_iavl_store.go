package store

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/log"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

// MultiReaderIAVLStore supports multiple concurrent readers more efficiently (in theory) than the
// original IAVLStore.
//
// LIMITATIONS:
// - Only the values from the leaf nodes of the latest saved IAVL tree are stored in valueDB,
//   which means MultiReaderIAVLStore can only load the latest IAVL tree. Rollback to an earlier
//   version is currently impossible.
type MultiReaderIAVLStore struct {
	IAVLStore
	valueDB    db.DBWrapper
	valueBatch dbm.Batch
}

func (s *MultiReaderIAVLStore) Delete(key []byte) {
	s.IAVLStore.Delete(key)
	s.valueBatch.Delete(key)
}

func (s *MultiReaderIAVLStore) Set(key, val []byte) {
	s.IAVLStore.Set(key, val)
	s.valueBatch.Set(key, val)
}

func (s *MultiReaderIAVLStore) Has(key []byte) bool {
	return s.IAVLStore.Has(key)
}

func (s *MultiReaderIAVLStore) Get(key []byte) []byte {
	return s.IAVLStore.Get(key)
}

func (s *MultiReaderIAVLStore) Range(prefix []byte) plugin.RangeData {
	return s.IAVLStore.Range(prefix)
}

func (s *MultiReaderIAVLStore) SaveVersion() ([]byte, int64, error) {
	hash, ver, err := s.IAVLStore.SaveVersion()
	if err != nil {
		return nil, 0, err
	}

	s.valueBatch.Write()
	s.valueBatch = s.valueDB.NewBatch()

	return hash, ver, nil
}

func (s *MultiReaderIAVLStore) GetSnapshot() Snapshot {
	return &multiReaderIAVLStoreSnapshot{
		Snapshot: s.valueDB.GetSnapshot(),
	}
}

func (s *MultiReaderIAVLStore) getValue(key []byte) []byte {
	// TODO: In theory the IAVL tree shouldn't try to load any key in s.valueBatch,
	//       but need to test what happens when Delete, Set, Delete, Set is called for the same
	//       key. Otherwise have to maintain a map of pending changes like similar to cacheTx.
	return s.valueDB.Get(key)
}

// NewMultiReaderIAVLStore creates a new MultiReaderIAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
func NewMultiReaderIAVLStore(
	nodeDB dbm.DB, valueDB db.DBWrapper, maxVersions int64,
) (*MultiReaderIAVLStore, error) {
	s := &MultiReaderIAVLStore{
		valueDB:    valueDB,
		valueBatch: valueDB.NewBatch(),
	}
	tree := iavl.NewMutableTreeWithExternalValueStore(nodeDB, 10000, s.getValue)
	// load the latest saved tree
	_, err := tree.LoadVersion(0)
	if err != nil {
		return nil, err
	}

	// always keep at least 2 of the last versions
	if (maxVersions != 0) && (maxVersions < 2) {
		maxVersions = 2
	}

	s.IAVLStore = IAVLStore{
		tree:        tree,
		maxVersions: maxVersions,
	}
	return s, nil
}

type multiReaderIAVLStoreSnapshot struct {
	db.Snapshot
}

func (s *multiReaderIAVLStoreSnapshot) Range(prefix []byte) plugin.RangeData {
	ret := make(plugin.RangeData, 0)
	it := s.Snapshot.NewIterator(prefix, prefixRangeEnd(prefix))
	defer it.Close()

	for ; it.Valid(); it.Next() {
		k, err := util.UnprefixKey(it.Key(), prefix)
		if err != nil {
			log.Error("failed to unprefix key", "key", it.Key(), "prefix", prefix, "err", err)
			panic(err)
		}
		re := &plugin.RangeEntry{
			Key:   k,
			Value: it.Value(),
		}
		ret = append(ret, re)
	}
	return ret
}
