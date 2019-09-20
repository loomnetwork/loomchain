package loomchain

import (
	"github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/go-loom/types"

	appstate "github.com/loomnetwork/loomchain/state"
)

type Transaction = types.Transaction

type TxRouter struct {
	deliverTxRoutes map[uint32]RouteHandler
	checkTxRoutes   map[uint32]RouteHandler
}

type RouteHandler func(txID uint32, state appstate.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error)

type RouteConditionFunc func(txID uint32, state appstate.State, txBytes []byte, isCheckTx bool) bool

var GeneratePassthroughRouteHandler = func(txHandler TxHandler) RouteHandler {
	return func(txID uint32, state appstate.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
		return txHandler.ProcessTx(state, txBytes, isCheckTx)
	}
}

func GenerateConditionalRouteHandler(conditionFn RouteConditionFunc, onTrue TxHandler, onFalse TxHandler) RouteHandler {
	return RouteHandler(func(txId uint32, state appstate.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
		if conditionFn(txId, state, txBytes, isCheckTx) {
			return onTrue.ProcessTx(state, txBytes, isCheckTx)
		}
		return onFalse.ProcessTx(state, txBytes, isCheckTx)
	})
}

func NewTxRouter() *TxRouter {
	return &TxRouter{
		deliverTxRoutes: make(map[uint32]RouteHandler),
		checkTxRoutes:   make(map[uint32]RouteHandler),
	}
}

func (r *TxRouter) HandleDeliverTx(txID uint32, handler RouteHandler) {
	if _, ok := r.deliverTxRoutes[txID]; ok {
		panic("handler for transaction already registered")
	}

	r.deliverTxRoutes[txID] = handler
}

func (r *TxRouter) HandleCheckTx(txID uint32, handler RouteHandler) {
	if _, ok := r.checkTxRoutes[txID]; ok {
		panic("handler for transaction already registered")
	}

	r.checkTxRoutes[txID] = handler
}

func (r *TxRouter) ProcessTx(state appstate.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
	var res TxHandlerResult

	var tx Transaction
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return res, err
	}

	var routeHandler RouteHandler
	if isCheckTx {
		routeHandler = r.checkTxRoutes[tx.Id]
	} else {
		routeHandler = r.deliverTxRoutes[tx.Id]
	}

	return routeHandler(tx.Id, state, tx.Data, isCheckTx)
}
