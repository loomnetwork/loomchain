package store

import (
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/db"
)

type EventStore interface {
	KVStore
}

type KVEventStore struct {
	db.DBWrapper
}

var _ EventStore = &KVEventStore{}

func (s *KVEventStore) Range(prefix []byte) plugin.RangeData {
	// To implement
	return nil
}

func NewKVEventStore(dbWrapper db.DBWrapper) *KVEventStore {
	return &KVEventStore{DBWrapper: dbWrapper}
}
