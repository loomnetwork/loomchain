package store

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

// MultiReaderIAVLStore supports multiple concurrent readers more efficiently (in theory) than the
// original IAVLStore.
type MultiReaderIAVLStore struct {
	IAVLStore
	valueDB    dbm.DB
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
	return &multiReaderIAVLStoreSnapshot{}
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
// targetVersion can be used to load any previously saved version of the store, if set to zero then
// the last version that was saved will be loaded.
func NewMultiReaderIAVLStore(
	nodeDB dbm.DB, valueDB dbm.DB, maxVersions, targetVersion int64,
) (*MultiReaderIAVLStore, error) {
	s := &MultiReaderIAVLStore{
		valueDB:    valueDB,
		valueBatch: valueDB.NewBatch(),
	}
	tree := iavl.NewMutableTreeWithExternalValueStore(nodeDB, 10000, s.getValue)
	_, err := tree.LoadVersion(targetVersion)
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

// TODO: hook this up to the MultiReaderIAVLStore.valueDB using LevelDB snapshots,
//       or read-only txs, whatever the DB backend supports
type multiReaderIAVLStoreSnapshot struct {
}

func (s *multiReaderIAVLStoreSnapshot) Get(key []byte) []byte {
	return nil
}

func (s *multiReaderIAVLStoreSnapshot) Range(prefix []byte) plugin.RangeData {
	return nil
}

func (s *multiReaderIAVLStoreSnapshot) Has(key []byte) bool {
	return false
}

func (s *multiReaderIAVLStoreSnapshot) Release() {
}
