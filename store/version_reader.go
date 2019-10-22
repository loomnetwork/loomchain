package store

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/pkg/errors"
)

type iAVLSnapshot struct {
	IAVLStore
	version int64
}

func newIAVLVersionReader(store IAVLStore, version int64) (Snapshot, error) {
	if !store.tree.VersionExists(version) {
		return nil, errors.Errorf("version %d missing", version)
	}
	return &iAVLSnapshot{
		IAVLStore: store,
		version:   version,
	}, nil
}

func (s *iAVLSnapshot) Get(key []byte) []byte {
	value, _, _ := s.IAVLStore.tree.GetVersionedWithProof(key, s.version)
	return value
}

func (s *iAVLSnapshot) Range(prefix []byte) plugin.RangeData {
	rangeData := s.IAVLStore.Range(prefix)
	var rtv plugin.RangeData
	for _, entry := range rangeData {
		if s.Has(entry.Key) {
			rtv = append(rtv, entry)
		}
	}
	return rtv
}

func (s *iAVLSnapshot) Has(key []byte) bool {
	value, _, _ := s.IAVLStore.tree.GetVersionedWithProof(key, s.version)
	return value != nil
}

func (s *iAVLSnapshot) Release() {}
