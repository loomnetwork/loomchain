package store

import (
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tmlibs/db"
)

type IAVLStore struct {
	tree *iavl.VersionedTree
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

func (s *IAVLStore) Hash() []byte {
	return s.tree.Hash()
}

func (s *IAVLStore) Version() int64 {
	return s.tree.Version64()
}

func (s *IAVLStore) SaveVersion() (int64, error) {
	_, version, err := s.tree.SaveVersion()
	return version, err
}

func NewIAVLStore(db dbm.DB) *IAVLStore {
	return &IAVLStore{
		tree: iavl.NewVersionedTree(db, 10000),
	}
}
