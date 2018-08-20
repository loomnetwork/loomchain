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

func newLoomStateDB(abm *evmAccountBalanceManager, root common.Hash, db state.Database) (*LoomStateDB, error) {
	sdb, err := state.New(root, db)
	if err != nil {
		return nil, err
	}
	return &LoomStateDB{
		StateDB: sdb,
		abm:     abm,
	}, nil
}

func (s *LoomStateDB) GetBalance(addr common.Address) *big.Int {
	return s.abm.GetBalance(addr)
}

// The EVM shouldn't be calling any of the functions below, the only way to manipulate an account
// balance is through a transfer between accounts via the ethcoin contract.

func (s *LoomStateDB) SubBalance(common.Address, *big.Int) {
	panic("not implemented")
}

func (s *LoomStateDB) AddBalance(common.Address, *big.Int) {
	panic("not implemented")
}

func (s *LoomStateDB) SetBalance(addr common.Address, amount *big.Int) {
	panic("not implemented")
}
