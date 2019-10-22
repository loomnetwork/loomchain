package store

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	pruneDuration         metrics.Histogram
	deleteVersionDuration metrics.Histogram
)

func init() {
	const namespace = "loomchain"
	const subsystem = "pruning_iavl_store"

	pruneDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "prune_duration",
			Help:       "How long PruningIAVLStore.prune() took to execute (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"})
	deleteVersionDuration = kitprometheus.NewSummaryFrom(
		stdprometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       "delete_version_duration",
			Help:       "How long it took to delete a single version from the IAVL store (in seconds)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"})
}

type PruningIAVLStoreConfig struct {
	MaxVersions   int64 // maximum number of versions to keep when pruning
	BatchSize     int64 // maximum number of versions to delete in each cycle
	FlushInterval int64 // number of versions before flushing to disk
	Interval      time.Duration
	Logger        *loom.Logger
}

// PruningIAVLStore is a specialized IAVLStore that has a background thread that periodically prunes
// old versions. It should only be used to prune old clusters, on new clusters nodes will delete
// a version each time they save a new one, so the background thread, and all the extra locking
// is unnecessary.
type PruningIAVLStore struct {
	store       *IAVLStore
	mutex       *sync.RWMutex
	oldestVer   int64
	maxVersions int64
	batchSize   int64
	batchCount  uint64
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

	store, err := NewIAVLStore(db, maxVersions, 0, cfg.FlushInterval)
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

		go s.loopWithInterval(s.prune, cfg.Interval)
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

func (s *PruningIAVLStore) GetSnapshot(version int64) Snapshot {
	// This isn't an actual snapshot obviously, and never will be, but lets pretend...
	return &pruningIAVLStoreSnapshot{
		PruningIAVLStore: s,
	}
}

func (s *PruningIAVLStore) prune() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var err error
	defer func(begin time.Time) {
		lvs := []string{"error", fmt.Sprint(err != nil)}
		pruneDuration.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

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
			if err = s.deleteVersion(i); err != nil {
				return errors.Wrapf(err, "failed to delete tree version %d", i)
			}
		}
		s.oldestVer++
	}

	s.batchCount++
	return nil
}

func (s *PruningIAVLStore) deleteVersion(ver int64) error {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"error", fmt.Sprint(err != nil)}
		deleteVersionDuration.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = s.store.tree.DeleteVersion(ver)
	return err
}

// runWithRecovery should run in a goroutine, it will ensure the given function keeps on running in
// a goroutine as long as it doesn't panic due to a runtime error.
//[MGC] I believe this function shouldn't be used as we should just fail fast if this breaks
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

type pruningIAVLStoreSnapshot struct {
	*PruningIAVLStore
}

func (s *pruningIAVLStoreSnapshot) Release() {
	// noop
}
