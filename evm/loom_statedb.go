// +build evm

package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// LoomStateDB overrides some of the state.StateDB functions used to manage ETH balances to allow
// EVM contracts to seamlessly access ETH balances through the ethcoin Go contract.
type LoomStateDB struct {
	*state.StateDB
	abm *evmAccountBalanceManager
}

func newLoomStateDB(abm *evmAccountBalanceManager, sdb *state.StateDB) (*LoomStateDB, error) {
	return &LoomStateDB{
		StateDB: sdb,
		abm:     abm,
	}, nil
}

func (s *LoomStateDB) GetBalance(addr common.Address) *big.Int {
	return s.abm.GetBalance(addr)
}

func (s *LoomStateDB) SubBalance(address common.Address, amount *big.Int) {
	s.abm.SubBalance(address, amount)
}

func (s *LoomStateDB) AddBalance(address common.Address, amount *big.Int) {
	s.abm.AddBalance(address, amount)
}

func (s *LoomStateDB) SetBalance(address common.Address, amount *big.Int) {
	s.abm.SetBalance(address, amount)
}

func (s *LoomStateDB) Suicide(address common.Address) bool {
	s.SetBalance(address, common.Big0)
	return s.StateDB.Suicide(address)
}
