package store

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

// MultiReaderIAVLStoreSnapshotVersion indicates the type of snapshot that should be created by
// MultiReaderIAVLStore.GetSnapshot()
type MultiReaderIAVLStoreSnapshotVersion int

const (
	// MultiReaderIAVLStoreSnapshotV1 Get/Has/Range go through the DB snapshot
	// implemented by multiReaderIAVLStoreDBSnapshot
	MultiReaderIAVLStoreSnapshotV1 MultiReaderIAVLStoreSnapshotVersion = 1
	// MultiReaderIAVLStoreSnapshotV2 Get/Has go through the IAVL tree, Range through the DB snapshot
	// implemented by multiReaderIAVLStoreHybridSnapshot
	MultiReaderIAVLStoreSnapshotV2 MultiReaderIAVLStoreSnapshotVersion = 2
	// MultiReaderIAVLStoreSnapshotV3 Get/Has/Range go through the IAVL tree
	// implemented by multiReaderIAVLStoreTreeSnapshot
	MultiReaderIAVLStoreSnapshotV3 MultiReaderIAVLStoreSnapshotVersion = 3
)

// NodeDBVersion indicates which iavl.NodeDB should be used by MultiReaderIAVLStore
type NodeDBVersion int

const (
	// NodeDBV1 corresponds to a single-mutex NodeDB
	NodeDBV1 NodeDBVersion = 1
	// NodeDBV2 corresponds to a multi-mutex NodeDB
	NodeDBV2 NodeDBVersion = 2
)

var (
	valueDBHeaderPrefix = []byte("dbh")
	valueDBVersionKey   = util.PrefixKey(valueDBHeaderPrefix, []byte("v"))
)

var _ = VersionedKVStore(&MultiReaderIAVLStore{})

// MultiReaderIAVLStore supports multiple concurrent readers more efficiently (in theory) than the
// original IAVLStore.
//
// The leaf nodes of the IAVL tree contain the actual values of the keys in the store, these leaf
// values are saved to a separate DB (valueDB/app_state.db) from the tree nodes themselves (app.db).
// Snapshots created by this store only access keys & values stored in the valueDB.
//
// LIMITATIONS:
// - Only the values from the leaf nodes of the latest saved IAVL tree are stored in valueDB,
//   which means MultiReaderIAVLStore can only load the latest IAVL tree. Rollback to an earlier
//   version is currently impossible.
// - Set/Delete/SaveVersion must be called from a single thread, i.e. there can only be one writer.
type MultiReaderIAVLStore struct {
	IAVLStore
	valueDB         db.DBWrapper
	valueBatch      dbm.Batch
	lastSavedTree   unsafe.Pointer // *iavl.ImmutableTree
	snapshotVersion MultiReaderIAVLStoreSnapshotVersion
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

	// Store the tree version in the value DB so it's possible to verify the two DBs are in sync
	// on startup.
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(ver))
	s.valueBatch.Set(valueDBVersionKey, buf)

	s.valueBatch.Write()
	s.valueBatch = s.valueDB.NewBatch()

	if err := s.setLastSavedTreeToVersion(ver); err != nil {
		return nil, 0, err
	}

	return hash, ver, nil
}

func (s *MultiReaderIAVLStore) GetSnapshot() Snapshot {
	switch s.snapshotVersion {
	case MultiReaderIAVLStoreSnapshotV1:
		return &multiReaderIAVLStoreDBSnapshot{
			Snapshot: s.valueDB.GetSnapshot(),
		}
	case MultiReaderIAVLStoreSnapshotV2:
		// TODO: It's possible that the DB snapshot is for block N while the tree is still for
		//       block N-1. Need to compare the tree version with the snapshot version, if they
		//       don't match then call atomic.LoadPointer(&s.lastSavedTree) again until the versions
		//       are in sync.
		tree := (*iavl.ImmutableTree)(atomic.LoadPointer(&s.lastSavedTree))
		return &multiReaderIAVLStoreHybridSnapshot{
			multiReaderIAVLStoreDBSnapshot: multiReaderIAVLStoreDBSnapshot{
				Snapshot: s.valueDB.GetSnapshot(),
			},
			ImmutableTree: tree,
		}
	case MultiReaderIAVLStoreSnapshotV3:
		tree := (*iavl.ImmutableTree)(atomic.LoadPointer(&s.lastSavedTree))
		return &multiReaderIAVLStoreTreeSnapshot{
			ImmutableTree: tree,
		}
	default:
		panic("Invalid snapshot version")
	}
}

func (s *MultiReaderIAVLStore) getValue(key []byte) []byte {
	// TODO: In theory the IAVL tree shouldn't try to load any key in s.valueBatch,
	//       but need to test what happens when Delete, Set, Delete, Set is called for the same
	//       key. Otherwise have to maintain a map of pending changes similar to cacheTx.
	return s.valueDB.Get(key)
}

func (s *MultiReaderIAVLStore) setLastSavedTreeToVersion(version int64) error {
	var err error
	var tree *iavl.ImmutableTree

	if version == 0 {
		tree = iavl.NewImmutableTree(nil, 0)
	} else {
		tree, err = s.IAVLStore.tree.GetImmutable(version)
		if err != nil {
			return errors.Wrapf(err, "failed to load immutable tree for version %v", version)
		}
	}

	atomic.StorePointer(&s.lastSavedTree, unsafe.Pointer(tree))
	return nil
}

// NewMultiReaderIAVLStore creates a new MultiReaderIAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
func NewMultiReaderIAVLStore(nodeDB dbm.DB, valueDB db.DBWrapper, cfg *AppStoreConfig) (*MultiReaderIAVLStore, error) {
	s := &MultiReaderIAVLStore{
		valueDB:         valueDB,
		valueBatch:      valueDB.NewBatch(),
		snapshotVersion: MultiReaderIAVLStoreSnapshotVersion(cfg.SnapshotVersion),
	}
	var ndb iavl.NodeDB
	switch cfg.NodeDBVersion {
	case NodeDBV1:
		ndb = iavl.NewNodeDB(nodeDB, cfg.NodeCacheSize, s.getValue)
	case NodeDBV2:
		ndb = iavl.NewNodeDB2(nodeDB, cfg.NodeCacheSize, s.getValue)
	default:
		return nil, errors.New("invalid AppStore.NodeDBVersion")
	}
	tree := iavl.NewMutableTreeWithNodeDB(ndb)
	// load the latest saved tree
	treeVer, err := tree.LoadVersion(0)
	if err != nil {
		return nil, err
	}

	// Verify the valueDB contains values for the current tree version, if the node crashed during
	// the last block commit the valueDB may not have been updated.
	var valueDBVer uint64
	if buf := valueDB.Get(valueDBVersionKey); buf != nil {
		valueDBVer = binary.BigEndian.Uint64(buf)
	}

	// TODO: Remove (valueDBVer > 0), it's a quick hack to get nodes running with an older
	//       app_state.db (that's missing valueDBVersionKey).
	// TODO: If treeVer + 1 == valueDBVer could rollback app.db to previous version using
	//       tree.LoadVersionForOverwriting(treeVer), that will force TM to replay the last block
	//       and bring the nodeDB back in sync with the valueDB (assuming the node doesn't crash).
	if (valueDBVer > 0) && (uint64(treeVer) != valueDBVer) {
		return nil, fmt.Errorf("nodeDB at version %d is out of sync with valueDB at version %d", treeVer, valueDBVer)
	}

	maxVersions := cfg.MaxVersions
	// always keep at least 2 of the last versions
	if (cfg.MaxVersions != 0) && (cfg.MaxVersions < 2) {
		maxVersions = 2
	}

	s.IAVLStore = IAVLStore{
		tree:        tree,
		maxVersions: maxVersions,
	}

	if err := s.setLastSavedTreeToVersion(treeVer); err != nil {
		return nil, err
	}

	return s, nil
}

// Get/Has/Range go through the DB snapshot
type multiReaderIAVLStoreDBSnapshot struct {
	db.Snapshot
}

func (s *multiReaderIAVLStoreDBSnapshot) Range(prefix []byte) plugin.RangeData {
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

// Get/Has go through an IAVL tree, Range through the DB snapshot
type multiReaderIAVLStoreHybridSnapshot struct {
	multiReaderIAVLStoreDBSnapshot // provides the Range implementation
	*iavl.ImmutableTree
}

func (s *multiReaderIAVLStoreHybridSnapshot) Get(key []byte) []byte {
	_, val := s.ImmutableTree.Get(key)
	return val
}

func (s *multiReaderIAVLStoreHybridSnapshot) Has(key []byte) bool {
	return s.ImmutableTree.Has(key)
}

// Get/Has/Range go through an IAVL tree
type multiReaderIAVLStoreTreeSnapshot struct {
	*iavl.ImmutableTree
}

func (s *multiReaderIAVLStoreTreeSnapshot) Get(key []byte) []byte {
	_, val := s.ImmutableTree.Get(key)
	return val
}

func (s *multiReaderIAVLStoreTreeSnapshot) Range(prefix []byte) plugin.RangeData {
	ret := make(plugin.RangeData, 0)
	keys, values, _, err := s.ImmutableTree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), 0)
	if err != nil {
		log.Error("failed to get range", "err", err)
		panic(err)
	}

	for i, x := range keys {
		k, err := util.UnprefixKey(x, prefix)
		if err != nil {
			log.Error("failed to unprefix key", "key", x, "prefix", prefix, "err", err)
			panic(err)
		}
		re := &plugin.RangeEntry{
			Key:   k,
			Value: values[i],
		}
		ret = append(ret, re)
	}

	return ret
}

func (s *multiReaderIAVLStoreTreeSnapshot) Release() {
	s.ImmutableTree = nil
}
