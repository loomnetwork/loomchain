// +build evm

package ethdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/loomnetwork/loomchain"
)

type EthDbManager interface {
	NewEthDb(loomchain.State, *EthdbLogContext) (ethdb.Database, error)
}

type EthDbHandler struct {
	v EthDbType
}

func NewEthDbManager(ethDbType EthDbType) EthDbManager {
	switch ethDbType {
	case EthDbNone:
		panic("EthDb required for evm builds")
	case EthDbLoom:
		return EthDbHandler{ v: ethDbType }
	case EthDbLdbDatabase:
		return EthDbHandler{ v: ethDbType }
	case EthDbMemDb:
		return EthDbHandler{ v: ethDbType }
	default:
		panic(fmt.Sprintf("unrecognised ethdb type %v", ethDbType))
	}
}

func (e EthDbHandler)  NewEthDb(state loomchain.State, ctx *EthdbLogContext) (ethdb.Database, error) {
	switch e.v {
	case EthDbNone:
		panic("EthDb required for evm builds")
	case EthDbLoom:
		return NewLoomEthdb(state, ctx), nil
	case EthDbLdbDatabase:
		return ethdb.NewLDBDatabase("evm.db",0, 0)
	case EthDbMemDb:
		return ethdb.NewMemDatabase(), nil
	default:
		panic(fmt.Sprintf("unrecognised ethdb type %v", e.v))
	}
}