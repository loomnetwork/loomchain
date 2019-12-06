package loomchain

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/loomnetwork/loomchain/store"
)

// EVMState contains the mutable EVM state.
type EVMState struct {
	sdb      *state.StateDB
	evmStore *store.EvmStore
}

// NewEVMState returns the EVM state corresponding to the current version of the given store.
func NewEVMState(evmStore *store.EvmStore) (*EVMState, error) {
	evmRoot, _ := evmStore.Version()
	sdb, err := state.New(common.BytesToHash(evmRoot), state.NewDatabaseWithTrieDB(evmStore.TrieDB()))
	if err != nil {
		return nil, err
	}
	return &EVMState{
		evmStore: evmStore,
		sdb:      sdb,
	}, nil
}

// Commit writes the state changes that occurred since the previous commit to the underlying store.
func (s *EVMState) Commit() error {
	evmStateRoot, err := s.sdb.Commit(true)
	if err != nil {
		return err
	}
	s.evmStore.SetCurrentRoot(evmStateRoot[:])
	// Clear out old state data such as logs and cache to free up memory
	s.sdb.Reset(evmStateRoot)
	return nil
}

// GetSnapshot returns the EVMState instance containing the state as it was at the given version.
// The specified root is expected to match the root of the returned state, if the roots don't match
// an error will be returned.
// NOTE: Do not call Commit on the returned instance.
func (s *EVMState) GetSnapshot(version int64, root []byte) (*EVMState, error) {
	r, v := s.evmStore.GetRootAt(version)
	if !bytes.Equal(r, root) {
		return nil, fmt.Errorf(
			"EVM roots mismatch, expected (%d): %X, actual (%d): %X",
			version, root, v, r,
		)
	}
	// The cachingDB instance created by state.NewDatabaseWithTrieDB() contains a codeSizeCache which
	// probably shouldn't be shared between the EVMState instance used by the tx handlers and the
	// snapshots instances used by the query server. Which is why NewDatabaseWithTrieDB() is used
	// here instead of s.sdb.Database().
	sdb, err := state.New(
		common.BytesToHash(r),
		state.NewDatabaseWithTrieDB(s.evmStore.TrieDB()),
	)
	if err != nil {
		return nil, err
	}
	return &EVMState{
		evmStore: nil, // this will ensure that Commit() will panic
		sdb:      sdb,
	}, nil
}

// Clone returns a copy of the EVMState instance.
// NOTE: Do not call Commit on the returned instance.
func (s *EVMState) Clone() *EVMState {
	return &EVMState{
		evmStore: nil, // this will ensure that Commit() will panic
		sdb:      s.sdb.Copy(),
	}
}

func (s *EVMState) StateDB() *state.StateDB {
	return s.sdb
}
