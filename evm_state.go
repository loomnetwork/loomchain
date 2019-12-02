package loomchain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/loomnetwork/loomchain/store"
)

// EVMState contains the mutable EVM state.
type EVMState struct {
	sdb      *state.StateDB
	evmStore *store.EvmStore
}

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
	s.evmStore.SetVMRootKey(evmStateRoot[:])
	// Clear out old state data such as logs and cache to free up memory
	s.sdb.Reset(evmStateRoot)
	return nil
}

// GetSnapshot returns the EVMState instance containing the state as it was at the given version.
// NOTE: Do not call Commit on the returned instance.
func (s *EVMState) GetSnapshot(version int64) (*EVMState, error) {
	stateDB, err := state.New(common.BytesToHash(s.evmStore.GetRootAt(version)), s.sdb.Database())
	if err != nil {
		return nil, err
	}
	return &EVMState{sdb: stateDB, evmStore: s.evmStore}, nil
}

func (s *EVMState) Clone() *EVMState {
	return &EVMState{
		evmStore: s.evmStore,
		sdb:      s.sdb.Copy(),
	}
}

func (s *EVMState) StateDB() *state.StateDB {
	return s.sdb
}
