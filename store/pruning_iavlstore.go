package store

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type PruningIAVLStoreConfig struct {
	MaxVersions int64 // maximum number of versions to keep when pruning
	BatchSize   int64 // maximum number of versions to delete in each cycle
	Interval    time.Duration
	Logger      *loom.Logger
}

// PruningIAVLStore is a specialized IAVLStore that has a background thread that periodically prunes
// old versions. It should only be used to prune old clusters, on new clusters nodes will delete
// a version each time they save a new one, so the background thread, and all the extra locking
// is unecessary.
type PruningIAVLStore struct {
	store       *IAVLStore
	mutex       *sync.RWMutex
	oldestVer   int64
	maxVersions int64
	batchSize   int64
	batchCount  uint64
	elapsedTime time.Duration
	logger      *loom.Logger
}

// NewPruningIAVLStore creates a new PruningIAVLStore.
// maxVersions can be used to specify how many versions should be retained, if set to zero then
// old versions will never been deleted.
func NewPruningIAVLStore(db dbm.DB, cfg PruningIAVLStoreConfig) (*PruningIAVLStore, error) {
	// always keep at least 2 of the latest versions
	maxVersions := cfg.MaxVersions
	if (maxVersions != 0) && (maxVersions < 2) {
		maxVersions = 2
	}

	store, err := NewIAVLStore(db, maxVersions, 0)
	if err != nil {
		return nil, err
	}

	s := &PruningIAVLStore{
		store:       store,
		mutex:       &sync.RWMutex{},
		maxVersions: maxVersions,
		batchSize:   cfg.BatchSize,
		logger:      cfg.Logger,
	}

	if s.logger == nil {
		s.logger = log.Default
	}

	if maxVersions != 0 {
		latestVer := store.Version()

		oldestVer := int64(0)
		if cfg.BatchSize > 1 {
			for i := int64(1); i <= latestVer; i++ {
				if store.tree.VersionExists(i) {
					oldestVer = i
					break
				}
			}
		}
		s.oldestVer = oldestVer

		go s.runWithRecovery(func() {
			s.loopWithInterval(s.prune, cfg.Interval)
		})
	}

	return s, nil
}

func (s *PruningIAVLStore) Delete(key []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.store.Delete(key)
}

func (s *PruningIAVLStore) Set(key, val []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.store.Set(key, val)
}

func (s *PruningIAVLStore) Has(key []byte) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.store.Has(key)
}

func (s *PruningIAVLStore) Get(key []byte) []byte {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.store.Get(key)
}

func (s *PruningIAVLStore) Range(prefix []byte) plugin.RangeData {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.store.Range(prefix)
}

func (s *PruningIAVLStore) Hash() []byte {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.store.Hash()
}

func (s *PruningIAVLStore) Version() int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.store.Version()
}

func (s *PruningIAVLStore) SaveVersion() ([]byte, int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	hash, ver, err := s.store.SaveVersion()
	if err == nil && s.oldestVer == 0 {
		s.oldestVer = ver
	}
	return hash, ver, err
}

func (s *PruningIAVLStore) Prune() error {
	// pruning is done in the goroutine, so do nothing here
	return nil
}

func (s *PruningIAVLStore) prune() error {
	startTime := time.Now()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	latestVer := s.store.Version()
	endVer := latestVer - s.maxVersions

	if (s.oldestVer == 0) || (s.oldestVer > endVer) {
		return nil // nothing to prune yet
	}

	if (endVer - s.oldestVer) > s.batchSize {
		endVer = s.oldestVer + s.batchSize
	}

	if endVer > (latestVer - 2) {
		endVer = latestVer - 2
	}

	for i := s.oldestVer; i <= endVer; i++ {
		if s.store.tree.VersionExists(i) {
			if err := s.store.tree.DeleteVersion(i); err != nil {
				return errors.Wrapf(err, "failed to delete tree version %d", i)
			}
		}
		s.oldestVer++
	}

	s.batchCount++
	s.elapsedTime += time.Since(startTime)

	if (s.batchCount % 500) == 0 {
		s.logger.Info(fmt.Sprintf("PruningIAVLStore: pruned %v batches in %v minutes",
			s.batchCount, s.elapsedTime.Minutes(),
		))
	}

	return nil
}

// runWithRecovery should run in a goroutine, it will ensure the given function keeps on running in
// a goroutine as long as it doesn't panic due to a runtime error.
func (s *PruningIAVLStore) runWithRecovery(run func()) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Recovered from panic in PruningIAVLStore goroutine", "r", r)
			// Unless it's a runtime error restart the goroutine
			if _, ok := r.(runtime.Error); !ok {
				time.Sleep(30 * time.Second)
				s.logger.Info("Restarting PruningIAVLStore goroutine...\n")
				go s.runWithRecovery(run)
			}
		}
	}()
	run()
}

// loopWithInterval will execute the step function in an endless loop, sleeping for the specified
// interval at the end of each loop iteration.
func (s *PruningIAVLStore) loopWithInterval(step func() error, interval time.Duration) {
	for {
		if err := step(); err != nil {
			s.logger.Error("PruneIAVLStore encountered an error", "err", err)
		}
		time.Sleep(interval)
	}
}
