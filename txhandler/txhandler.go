package txhandler

import (
	"github.com/ethereum/go-ethereum/core/vm"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"

	"github.com/loomnetwork/loomchain/state"
)

type TxHandler interface {
	ProcessTx(state.State, []byte, bool) (TxHandlerResult, error)
}

type TxHandlerFunc func(state.State, []byte, bool) (TxHandlerResult, error)

type TxHandlerResult struct {
	Data             []byte
	ValidatorUpdates []abci.Validator
	Info             string
	// Tags to associate with the tx that produced this result. Tags can be used to filter txs
	// via the ABCI query interface (see https://godoc.org/github.com/tendermint/tendermint/libs/pubsub/query)
	Tags []common.KVPair
}

func (f TxHandlerFunc) ProcessTx(s state.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
	return f(s, txBytes, isCheckTx)
}

type TxHandlerFactory interface {
	TxHandler(tracer vm.Tracer) (TxHandler, error)
}
