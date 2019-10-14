package store

import (
	"bytes"

	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
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
	return value != nil
}

type multiWriterVersionReader struct {
	appStore KVReader
	evmStore KVReader
	version  int64
}

func newMultiWriterVersionReader(store MultiWriterAppStore, version int64) (KVReader, error) {
	if !store.appStore.tree.VersionExists(version) {
		return nil, errors.Errorf("version %d missing", version)
	}
	appStore, err := newIAVLVersionReader(*store.appStore, version)
	if err != nil {
		return nil, err
	}
	evmStore := NewEvmStore(store.evmStore.evmDB, 100)
	if err := evmStore.LoadVersion(version); err != nil {
		return nil, err
	}
	return &multiWriterVersionReader{
		appStore: appStore,
		evmStore: evmStore,
		version:  version,
	}, nil
}

func (m *multiWriterVersionReader) Get(key []byte) []byte {
	if util.HasPrefix(key, vmPrefix) {
		return m.evmStore.Get(key)
	}
	return m.appStore.Get(key)
}

// Range returns a range of keys
func (m *multiWriterVersionReader) Range(prefix []byte) plugin.RangeData {
	if len(prefix) == 0 {
		panic(errors.New("Range over nil prefix not implemented"))
	}

	if bytes.Equal(prefix, vmPrefix) || util.HasPrefix(prefix, vmPrefix) {
		return m.evmStore.Range(prefix)
	}
	return m.appStore.Range(prefix)
}

// Has checks if a key exists.
func (m *multiWriterVersionReader) Has(key []byte) bool {
	if util.HasPrefix(key, vmPrefix) {
		return m.evmStore.Has(key)
	}
	return m.appStore.Has(key)
}
