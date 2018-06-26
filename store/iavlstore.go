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

func (s *IAVLStore) All() ([][]byte, [][]byte, error) {
	b := byte(255)
	keys, values, _, err := s.tree.Tree().GetRangeWithProof([]byte{0}, []byte{b}, 0)
	return keys, values, err
}

func (s *IAVLStore) Hash() []byte {
	return s.tree.Hash()
}

func (s *IAVLStore) Version() int64 {
	return s.tree.Version64()
}

func (s *IAVLStore) SaveVersion() ([]byte, int64, error) {
	return s.tree.SaveVersion()
}

func NewIAVLStore(db dbm.DB) (*IAVLStore, error) {
	tree := iavl.NewVersionedTree(db, 10000)
	_, err := tree.Load()
	if err != nil {
		return nil, err
	}
	return &IAVLStore{
		tree: tree,
	}, nil
}
