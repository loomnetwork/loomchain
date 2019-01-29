package loomchain

import (
	"github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/go-loom/types"
)

type Transaction = types.Transaction

type TxRouter struct {
	deliverTxRoutes map[uint32]TxHandler
	checkTxRoutes   map[uint32]TxHandler
}

func NewTxRouter() *TxRouter {
	return &TxRouter{
		deliverTxRoutes: make(map[uint32]TxHandler),
		checkTxRoutes:   make(map[uint32]TxHandler),
	}
}

func (r *TxRouter) HandleDeliverTx(txID uint32, handler TxHandler) {
	if _, ok := r.deliverTxRoutes[txID]; ok {
		panic("handler for transaction already registered")
	}

	r.deliverTxRoutes[txID] = handler
}

func (r *TxRouter) HandleCheckTx(txID uint32, handler TxHandler) {
	if _, ok := r.checkTxRoutes[txID]; ok {
		panic("handler for transaction already registered")
	}

	r.checkTxRoutes[txID] = handler
}

func (r *TxRouter) ProcessTx(state State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
	var res TxHandlerResult

	var tx Transaction
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return res, err
	}

	var handler TxHandler
	if isCheckTx {
		handler = r.checkTxRoutes[tx.Id]
	} else {
		handler = r.deliverTxRoutes[tx.Id]
	}

	return handler.ProcessTx(state, tx.Data, isCheckTx)
}
