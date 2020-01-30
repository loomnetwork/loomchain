package store

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	lru "github.com/hashicorp/golang-lru"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	defaultRoot = []byte{1}
	rootHashKey = util.PrefixKey(vmPrefix, rootKey)
	// Prefix for versioned Patricia roots
	evmRootPrefix = []byte("evmroot")

	commitDuration metrics.Histogram
)

func init() {
	commitDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  "loomchain",
			Subsystem:  "evmstore",
			Name:       "commit",
			Help:       "How long EvmStore.Commit() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{},
	)
}

func evmRootKey(blockHeight int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(blockHeight))
	return util.PrefixKey(vmPrefix, []byte(evmRootPrefix), b)
}

func getVersionFromEvmRootKey(key []byte) (int64, error) {
	v, err := util.UnprefixKey(key, util.PrefixKey(vmPrefix, []byte(evmRootPrefix)))
	if err != nil {
		return 0, err
	}
	version := int64(binary.BigEndian.Uint64(v))
	return version, nil
}

// EvmStore persists EVM state to a DB.
type EvmStore struct {
	evmDB         db.DBWrapper
	rootHash      []byte
	lastSavedRoot []byte
	rootCache     *lru.Cache
	version       int64
	trieDB        *trie.Database
	flushInterval int64
}

// NewEvmStore returns a new instance of the store backed by the given DB.
func NewEvmStore(evmDB db.DBWrapper, numCachedRoots int, flushInterval int64) *EvmStore {
	rootCache, err := lru.New(numCachedRoots)
	if err != nil {
		panic(err)
	}
	evmStore := &EvmStore{
		evmDB:         evmDB,
		rootCache:     rootCache,
		flushInterval: flushInterval,
	}
	ethDB := NewLoomEthDB(evmDB)
	evmStore.trieDB = trie.NewDatabase(ethDB)
	return evmStore
}

// Commit may persist the changes made to the store since the last commit to the underlying DB.
// The specified version is associated with the current root, which is returned by this function.
// Whether or not changes are actually flushed to the DB depends on the flush interval, which can
// be specified when calling NewEvmStore(), and overriden via the flushIntervalOverride parameter
// when calling Commit() iff the store was created with flushInterval == 0.
func (s *EvmStore) Commit(version, flushIntervalOverride int64) []byte {
	defer func(begin time.Time) {
		commitDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())

	currentRoot := make([]byte, len(s.rootHash))
	copy(currentRoot, s.rootHash)
	// default root is an indicator for empty root
	if bytes.Equal(currentRoot, []byte{}) {
		currentRoot = defaultRoot
	}

	flushInterval := s.flushInterval
	if flushInterval == 0 {
		flushInterval = flushIntervalOverride
	} else if flushInterval == -1 {
		flushInterval = 0
	}

	// Only commit Patricia tree every N blocks
	// TODO: What happens to all the roots that don't get committed? Are they just going to accumulate
	//       in the trie.Database.nodes cache forever?
	if flushInterval == 0 || version%flushInterval == 0 {
		// If the root hasn't changed since the last call to Commit that means no new state changes
		// occurred in the trie DB since then, so we can skip committing.
		if !bytes.Equal(defaultRoot, currentRoot) && !bytes.Equal(currentRoot, s.lastSavedRoot) {
			// trie.Database.Commit will call NewBatch (indirectly) to batch writes to evmDB
			if err := s.trieDB.Commit(common.BytesToHash(currentRoot), false); err != nil {
				panic(err)
			}
		}

		// We don't commit empty root but we need to save default root ([]byte{1}) as a placeholder of empty root
		// So the node won't get EVM root mismatch during the EVM root checking
		if !bytes.Equal(currentRoot, s.lastSavedRoot) {
			s.evmDB.Set(evmRootKey(version), currentRoot)
			s.lastSavedRoot = currentRoot
		}
	}

	s.rootCache.Add(version, currentRoot)
	s.version = version
	return currentRoot
}

func (s *EvmStore) LoadVersion(targetVersion int64) error {
	// find the last saved root
	root, version := s.getLastSavedRoot(targetVersion)
	if bytes.Equal(root, defaultRoot) {
		root = []byte{}
	}
	s.rootCache.Add(targetVersion, root)

	// nil root indicates that latest saved root below target version is not found
	if root == nil && targetVersion > 0 {
		return errors.Errorf("failed to load EVM root for version %d", targetVersion)
	}

	s.rootHash = root
	s.lastSavedRoot = root
	s.version = version
	return nil
}

func (s *EvmStore) Version() ([]byte, int64) {
	return s.rootHash, s.version
}

func (s *EvmStore) TrieDB() *trie.Database {
	return s.trieDB
}

// SetCurrentRoot sets the current EVM state root, this root must exist in the current trie DB.
// NOTE: This function must be called prior to each call to Commit.
// TODO: This is clunky, the root should just be passed into Commit!
func (s *EvmStore) SetCurrentRoot(root []byte) {
	s.rootHash = root
}

// getLastSavedRoot retrieves the EVM state root from disk that best matches the given version.
// The roots are not written to disk for every version, they only get written out when they change
// between versions, and even then depending on the flush interval some roots won't be written to disk.
func (s *EvmStore) getLastSavedRoot(targetVersion int64) ([]byte, int64) {
	start := util.PrefixKey(vmPrefix, evmRootPrefix)
	end := prefixRangeEnd(evmRootKey(targetVersion))
	iter := s.evmDB.ReverseIterator(start, end)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		version, err := getVersionFromEvmRootKey(iter.Key())
		if err != nil {
			return nil, 0
		}
		if version <= targetVersion || targetVersion == 0 {
			return iter.Value(), version
		}
	}
	return nil, 0
}

// GetRootAt returns the EVM state root corresponding to the given version.
// The second return value is version of the EVM state that corresponds to the returned root,
// it may be less than the version requested due to the reasons mentioned in getLastSavedRoot.
func (s *EvmStore) GetRootAt(version int64) ([]byte, int64) {
	// Expect cache to be almost 100% hit since cache miss yields extremely poor performance.
	// There's an assumption here that the cache will almost always contain all the in-mem-only
	// roots that haven't been flushed to disk yet, in the rare case where such a root is evicted
	// from the cache the last root persisted to disk will be returned instead. This means it's
	// possible (though highly unlikely) for queries to return stale state (since they rely on
	// snapshots corresponding to specific versions). This could be fixed by storing the in-mem-only
	// roots in another map instead of, or in addition to the cache.
	val, exist := s.rootCache.Get(version)
	if exist {
		return val.([]byte), version
	}
	return s.getLastSavedRoot(version)
}
