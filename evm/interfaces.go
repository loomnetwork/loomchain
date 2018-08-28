package evm

import (
	"github.com/loomnetwork/go-loom"
)

// AccountBalanceManager can be implemented to override the builtin account balance management in the EVM.
type AccountBalanceManager interface {
	GetBalance(addr loom.Address) (*loom.BigUInt, error)
	Transfer(from, to loom.Address, amount *loom.BigUInt) error
	AddBalance(addr loom.Address, amount *loom.BigUInt) error
	SubBalance(addr loom.Address, amount *loom.BigUInt) error
	SetBalance(addr loom.Address, amount *loom.BigUInt) error
}

type AccountBalanceManagerFactoryFunc func(readOnly bool) AccountBalanceManager
