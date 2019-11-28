package loomchain

import (
	gcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	gstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/loomnetwork/loomchain/store"
)

type EVMState struct {
	sdb      *gstate.StateDB
	evmStore *store.EvmStore
}

func NewEVMState(evmStore *store.EvmStore) (*EVMState, error) {
	ethDB := store.NewLoomEthDB(evmStore, nil)
	stateDB := state.NewDatabase(ethDB).WithTrieDB(evmStore.TrieDB())
	evmRoot, _ := evmStore.Version()
	sdb, err := state.New(gcommon.BytesToHash(evmRoot), stateDB)
	if err != nil {
		return nil, err
	}
	return &EVMState{
		evmStore: evmStore,
		sdb:      sdb,
	}, nil
}

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

func (s *EVMState) GetSnapshot(version int64) (*EVMState, error) {
	stateDB, err := gstate.New(gcommon.BytesToHash(s.evmStore.GetRootAt(version)), s.sdb.Database())
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

func (s *EVMState) StateDB() *gstate.StateDB {
	return s.sdb
}
