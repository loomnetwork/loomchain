package store

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/pkg/errors"
)

type iAVLVersionReader struct {
	IAVLStore
	version int64
}

func newIAVLVersionReader(store IAVLStore, version int64) (KVReader, error) {
	if !store.tree.VersionExists(version) {
		return nil, errors.Errorf("version %d missing", version)
	}
	return &iAVLVersionReader{
		IAVLStore: store,
		version:   version,
	}, nil
}

func (s *iAVLVersionReader) Get(key []byte) []byte {
	value, _, _ := s.IAVLStore.tree.GetVersionedWithProof(key, s.version)
	return value
}

// Range returns a range of keys
func (s *iAVLVersionReader) Range(prefix []byte) plugin.RangeData {
	rangeData := s.IAVLStore.Range(prefix)
	var rtv plugin.RangeData
	for _, entry := range rangeData {
		if s.Has(entry.Key) {
			rtv = append(rtv, entry)
		}
	}
	return rtv
}

// Has checks if a key exists.
func (s *iAVLVersionReader) Has(key []byte) bool {
	value, _, _ := s.IAVLStore.tree.GetVersionedWithProof(key, s.version)
	return value == nil
}
