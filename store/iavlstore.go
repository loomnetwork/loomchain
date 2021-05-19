package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	cdb "github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	pruneTime               metrics.Histogram
	iavlSaveVersionDuration metrics.Histogram
)

func init() {
	const namespace = "loomchain"
	const subsystem = "iavl_store"

	pruneTime = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "prune_duration",
			Help:       "How long IAVLStore.Prune() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"},
	)
	iavlSaveVersionDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "save_version",
			Help:       "How long IAVLStore.SaveVersion() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{},
	)
}

type IAVLStore struct {
	tree          *iavl.MutableTree
	maxVersions   int64 // maximum number of versions to keep when pruning
	flushInterval int64 // how often we persist to disk
	clonedDB      dbm.DB
}

func (s *IAVLStore) Delete(key []byte) {
	s.tree.Remove(key)
}

func (s *IAVLStore) Set(key, val []byte) {
	s.tree.Set(key, val)
}

func (s *IAVLStore) Has(key []byte) bool {
	return s.tree.Has(key)
}

func (s *IAVLStore) Get(key []byte) []byte {
	_, val := s.tree.Get(key)
	return val
}

// Returns the bytes that mark the end of the key range for the given prefix.
func prefixRangeEnd(prefix []byte) []byte {
	if prefix == nil {
		return nil
	}

	end := make([]byte, len(prefix))
	copy(end, prefix)

	for {
		if end[len(end)-1] != byte(255) {
			end[len(end)-1]++
			break
		} else if len(end) == 1 {
			end = nil
			break
		}
		end = end[:len(end)-1]
	}
	return end
}

func (s *IAVLStore) Range(prefix []byte) plugin.RangeData {
	return s.RangeWithLimit(prefix, 0)
}

// RangeWithLimit will return a list of keys & values that are prefixed by the given bytes (with a
// zero byte separator between the prefix and the key).
//
// If the limit is zero all matching keys will be returned, if the limit is greater than zero at most
// that many keys will be returned. Unfortunately, specifying a non-zero limit can result in somewhat
// unpredictable results, if there are N matching keys, and the limit is N, the number of keys
// returned may be less than N.
func (s *IAVLStore) RangeWithLimit(prefix []byte, limit int) plugin.RangeData {
	ret := make(plugin.RangeData, 0)

	keys, values, _, err := s.tree.GetRangeWithProof(prefix, prefixRangeEnd(prefix), limit)
	if err != nil {
		log.Error("failed to get range", "err", err)
		return ret
	}
	for i, k := range keys {
		// Tree range gives all keys that has prefix but it does not check zero byte
		// after the prefix. So we have to check zero byte after prefix using util.HasPrefix
		if util.HasPrefix(k, prefix) {
			k, err = util.UnprefixKey(k, prefix)
			if err != nil {
				log.Error("failed to unprefix key", "key", k, "prefix", prefix, "err", err)
				k = nil
			}

		} else {
			continue // Skip this key as it does not have the prefix
		}

		re := &plugin.RangeEntry{
			Key:   k,
			Value: values[i],
		}
		ret = append(ret, re)
	}

	return ret
}

func (s *IAVLStore) Hash() []byte {
	return s.tree.Hash()
}

func (s *IAVLStore) Version() int64 {
	return s.tree.Version()
}

func (s *IAVLStore) SaveVersion() ([]byte, int64, error) {
	var err error
	defer func(begin time.Time) {
		iavlSaveVersionDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())

	oldVersion := s.Version()
	flushInterval := s.flushInterval

	// TODO: Rather than loading the on-chain config here the flush interval override should be passed
	//       in as a parameter to SaveVersion().
	cfg, err := LoadOnChainConfig(s)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to load on-chain config")
	}

	if flushInterval == 0 {
		if cfg.GetAppStore().GetIAVLFlushInterval() != 0 {
			flushInterval = int64(cfg.GetAppStore().GetIAVLFlushInterval())
		}
	} else if flushInterval == -1 {
		flushInterval = 0
	}

	var version int64
	var hash []byte
	// Every X versions we should persist to disk
	flushToDisk := flushInterval == 0 || ((oldVersion+1)%flushInterval == 0)
	if flushToDisk {
		if flushInterval != 0 {
			log.Info("[IAVLStore] Flushing mem to disk", "version", oldVersion+1)
			hash, version, err = s.tree.FlushMemVersionDisk()
		} else {
			hash, version, err = s.tree.SaveVersion()
		}
	} else {
		hash, version, err = s.tree.SaveVersionMem()
	}

	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to save tree version %d", oldVersion+1)
	}

	// check if this is the block/version at which we must switch over to a new app.db
	cloneAtHeight := int64(cfg.GetAppStore().GetCloneStateAtHeight())
	if (cloneAtHeight > 0) && (cloneAtHeight == version) {
		// NOTE: Skip cloning if...
		// - The version being cloned hasn't been flushed to disk, this is done as a precaution
		//   because cloning of in-memory versions hasn't been tested.
		// - The destination DB isn't set, which means that the cloning & switchover has already
		//   happened, and shouldn't be done again because cloning to a DB that isn't empty is
		//   probably a bad idea.
		if !flushToDisk || (s.clonedDB == nil) {
			log.Warn(
				"[IAVLStore] Skipping cloning state",
				"version", version,
				"flushed", flushToDisk,
				"dbSet", s.clonedDB != nil,
			)
			return hash, version, nil
		}
		hash, version, err = s.cloneStateToDB()
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed to clone tree version %d", version)
		}
	}

	return hash, version, nil
}

func (s *IAVLStore) Prune() error {
	// keep all the versions
	if s.maxVersions == 0 {
		return nil
	}

	latestVer := s.Version()
	oldVer := latestVer - s.maxVersions
	if oldVer < 1 {
		return nil
	}

	var err error
	defer func(begin time.Time) {
		lvs := []string{"error", fmt.Sprint(err != nil)}
		pruneTime.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	if s.tree.VersionExists(oldVer) {
		if err = s.tree.DeleteVersion(oldVer); err != nil {
			return errors.Wrapf(err, "failed to delete tree version %d", oldVer)
		}
	}
	return nil
}

func (s *IAVLStore) GetSnapshot() Snapshot {
	// This isn't an actual snapshot obviously, and never will be, but lets pretend...
	return &iavlStoreSnapshot{
		IAVLStore: s,
	}
}

var (
	standardStorePrefixes = [][]byte{
		[]byte("nonce"),
		[]byte("feature"),
		[]byte("registry"),
		[]byte("reg_caddr"),
		[]byte("reg_crec"),
		[]byte("migrationId"),
	}
	standardStoreKeys = [][]byte{
		[]byte("config"),
		[]byte("minbuild"),
		[]byte("vmroot"),
	}
	// names of native contracts that can be resolved to an address via the contract registry
	nativeContractNames = []string{
		"addressmapper",
		"coin",
		"ethcoin",
		"dpos",
		"dposV2",
		"dposV3",
		"gateway",
		"loomcoin-gateway",
		"tron-gateway",
		"binance-gateway",
		"bsc-gateway",
		"deployerwhitelist",
		"user-deployer-whitelist",
		"chainconfig",
		"karma",
		"plasmacash",
	}
)

func (s *IAVLStore) getContractStorePrefix(contractName string) ([]byte, error) {
	contractAddrKeyPrefix := []byte("reg_caddr") // registry v2
	data := s.Get(util.PrefixKey(contractAddrKeyPrefix, []byte(contractName)))
	if len(data) == 0 {
		return nil, nil // contract is probably not deployed on this chain
	}
	var contractAddrPB types.Address
	if err := proto.Unmarshal(data, &contractAddrPB); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal contract address")
	}
	contractAddr := loom.UnmarshalAddressPB(&contractAddrPB)
	return util.PrefixKey([]byte("contract"), []byte(contractAddr.Local)), nil
}

func (s *IAVLStore) cloneStateToDB() ([]byte, int64, error) {
	// The version of the new tree will be incremented by one before it's saved to disk, so
	// to retain the same version number as the original tree when the new tree is saved to
	// disk we have to initialize the new tree with a lower version than the original.
	oldTree := s.tree
	newTree := iavl.NewMutableTreeWithVersion(s.clonedDB, 10000, oldTree.Version()-1)

	log.Info(
		"[IAVLStore] Started cloning state",
		"treeHeight", oldTree.Height(),
		"treeSize", oldTree.Size(),
		"treeVersion", oldTree.Version(),
		"treeHash", fmt.Sprintf("%x", oldTree.Hash()),
	)

	prefixes := append([][]byte{}, standardStorePrefixes...)
	// each native contract has its own prefix in the store
	for _, contractName := range nativeContractNames {
		prefix, err := s.getContractStorePrefix(contractName)
		if err != nil {
			return nil, 0, err
		}
		if prefix != nil {
			log.Debug(
				"[IAVLStore] Resolved contract store prefix",
				"contract", contractName,
				"prefix", fmt.Sprintf("%x", prefix),
			)
			prefixes = append(prefixes, prefix)
		}
	}

	numKeys := uint64(0)
	startTime := time.Now()

	// copy out all the keys under the prefixes that are still in use
	for _, prefix := range prefixes {
		var itError *error
		oldTree.IterateRange(
			prefix,
			prefixRangeEnd(prefix),
			true,
			func(key, value []byte) bool {
				// This is just a sanity check, should never actually happen!
				if !util.HasPrefix(key, prefix) {
					err := errors.Errorf(
						"key does not have prefix, skipped key: %x prefix: %x",
						key, prefix,
					)
					itError = &err
					return true // stop iteration
				}

				newTree.Set(key, value)
				numKeys++
				return false
			},
		)
		if itError != nil {
			return nil, 0, *itError
		}
	}

	// copy out the misc keys
	for _, key := range standardStoreKeys {
		if oldTree.Has(key) {
			_, value := oldTree.Get(key)
			newTree.Set(key, value)
			numKeys++
		}
	}

	hash, version, err := newTree.SaveVersion()
	if err != nil {
		return nil, version, errors.Wrap(err, "failed to save cloned tree")
	}
	log.Info(
		"[IAVLStore] Finished cloning state",
		"keys", numKeys,
		"mins", time.Since(startTime).Minutes(),
		"treeHeight", newTree.Height(),
		"treeSize", newTree.Size(),
		"treeVersion", newTree.Version(),
		"treeHash", fmt.Sprintf("%x", hash),
	)

	s.tree = newTree // discard the old tree
	s.clonedDB = nil // ensure cloning can't be repeated by accident
	return hash, version, nil
}

// NewIAVLStore creates a new IAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
// targetVersion can be used to load any previously saved version of the store, if set to zero then
// the last version that was saved will be loaded.
// flushInterval specifies the number of IAVL tree versions that should be kept in memory before
// writing a new version to disk. If set to zero every version will be written to disk unless overriden
// via the on-chain config. If set to -1 every version will always be written to disk, regardless of
// the on-chain config.
func NewIAVLStore(db dbm.DB, maxVersions, targetVersion, flushInterval int64) (*IAVLStore, error) {
	if flushInterval < -1 {
		return nil, errors.New("invalid flush interval")
	}

	tree := iavl.NewMutableTree(db, 10000)
	_, err := tree.LoadVersion(targetVersion)
	if err != nil {
		return nil, err
	}

	// always keep at least 2 of the last versions
	if (maxVersions != 0) && (maxVersions < 2) {
		maxVersions = 2
	}

	return &IAVLStore{
		tree:          tree,
		maxVersions:   maxVersions,
		flushInterval: flushInterval,
	}, nil
}

// NewIAVLStoreWithDualDBs creates a new IAVLStore.
// Initially the store will use the original DB and may switch over to the cloned DB when triggered
// by the CloneStateAtHeight on-chain config setting.
func NewIAVLStoreWithDualDBs(
	originalDB dbm.DB, clonedDB dbm.DB, maxVersions, targetVersion, flushInterval int64,
) (*IAVLStore, error) {
	store, err := NewIAVLStore(originalDB, maxVersions, targetVersion, flushInterval)
	if err != nil {
		return nil, err
	}
	store.clonedDB = clonedDB
	return store, nil
}

// Replace the old app.db (firstDBName) with the cloned app.db (secondDBName),
// but only if the cloned one contains the most recent state / version.
func SwitchToLatestAppStoreDB(dbBackend, firstDBName, secondDBName, directory string) error {
	if _, err := os.Stat(filepath.Join(directory, secondDBName+".db")); err != nil {
		if os.IsNotExist(err) {
			return nil // cloned app.db doesn't exist yet, so there's no need to do anything
		}
	}

	var firstDB, secondDB dbm.DB
	var err error
	switch dbBackend {
	case cdb.GoLevelDBBackend:
		firstDB, err = cdb.LoadReadOnlyGoLevelDB(firstDBName, directory)
		if err != nil {
			return err
		}
		secondDB, err = cdb.LoadReadOnlyGoLevelDB(secondDBName, directory)
		if err != nil {
			return err
		}
	case cdb.CLevelDBBackend:
		firstDB, err = cdb.LoadCLevelDB(firstDBName, directory)
		if err != nil {
			return err
		}
		secondDB, err = cdb.LoadCLevelDB(secondDBName, directory)
		if err != nil {
			return err
		}
	default:
		return nil
	}

	firstTree := iavl.NewMutableTree(firstDB, 0)
	firstTreeVersion, err := firstTree.LazyLoadVersion(0) // load latest version of the tree
	if err != nil {
		return errors.Wrap(err, "[IAVLStore] failed to load latest IAVL tree version from original DB")
	}
	secondTree := iavl.NewMutableTree(secondDB, 0)
	secondTreeVersion, err := secondTree.LazyLoadVersion(0)
	if err != nil {
		return errors.Wrap(err, "[IAVLStore] failed to load latest IAVL tree version from cloned DB")
	}

	firstDB.Close()
	secondDB.Close()

	if (secondTreeVersion > 0) && (secondTreeVersion >= firstTreeVersion) {
		// app.db -> old_app_<timestamp>.db
		srcName := firstDBName + ".db"
		destName := fmt.Sprintf("old_%s_%d.db", firstDBName, time.Now().Unix())
		log.Info("[IAVLStore] Moving app DB", "src", srcName, "dest", destName)
		if err := os.Rename(filepath.Join(directory, srcName), filepath.Join(directory, destName)); err != nil {
			return errors.Wrapf(err, "[IAVLStore] failed to move %s -> %s", srcName, destName)
		}
		// app_v2.db -> app.db
		srcName = secondDBName + ".db"
		destName = firstDBName + ".db"
		log.Info("[IAVLStore] Moving app DB", "src", srcName, "dest", destName)
		if err := os.Rename(filepath.Join(directory, srcName), filepath.Join(directory, destName)); err != nil {
			return errors.Wrapf(err, "[IAVLStore] failed to move %s -> %s", srcName, destName)
		}
	}
	return nil
}

type iavlStoreSnapshot struct {
	*IAVLStore
}

func (s *iavlStoreSnapshot) Release() {
	// noop
}
