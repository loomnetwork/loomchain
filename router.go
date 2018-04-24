package loom

import (
	"github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/loom-plugin/types"
)

type Transaction = types.Transaction

type TxRouter struct {
	routes map[uint32]TxHandler
}

func NewTxRouter() *TxRouter {
	return &TxRouter{
		routes: make(map[uint32]TxHandler),
	}
}

func (r *TxRouter) Handle(txID uint32, handler TxHandler) {
	if _, ok := r.routes[txID]; ok {
		panic("handler for transaction already registered")
	}

	r.routes[txID] = handler
}

func (r *TxRouter) ProcessTx(state State, txBytes []byte) (TxHandlerResult, error) {
	var res TxHandlerResult

	var tx Transaction
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return res, err
	}

	handler := r.routes[tx.Id]
	return handler.ProcessTx(state, tx.Data)
}
