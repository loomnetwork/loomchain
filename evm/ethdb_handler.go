// +build evm

package evm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/loomnetwork/loomchain"
)

type EthDbManager interface {
	NewEthDb(loomchain.State, *ethdbLogContext) (ethdb.Database, error)
}

type EthDbHandler struct {
	v      EthDbType
}

func NewEthDbHandler(ethDbType EthDbType) EthDbManager {
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

func (e EthDbHandler)  NewEthDb(state loomchain.State, ctx *ethdbLogContext) (ethdb.Database, error) {
	switch e.v {
	case EthDbNone:
		panic("EthDb required for evm builds")
	case EthDbLoom:
		return NewLoomEthdb(state, ctx), nil
	case EthDbLdbDatabase:
		return ethdb.NewLDBDatabase("ethdb",0, 0)
	case EthDbMemDb:
		return ethdb.NewMemDatabase(), nil
	default:
		panic(fmt.Sprintf("unrecognised ethdb type %v", e.v))
	}
}