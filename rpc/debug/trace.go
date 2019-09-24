// +build evm

package debug

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethapi"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/txhandler/middleware"
)

func TraceTransaction(
	app middleware.InMemoryApp,
	blockstore store.BlockStore,
	blockNumber int64,
	txIndex uint64,
	config eth.TraceConfig,
) (interface{}, error) {
	block, err := blockstore.GetBlockByHeight(&blockNumber)
	if err != nil {
		return nil, errors.Wrapf(err, "getting block information at height %v", blockNumber)
	}

	for i := uint64(0); i < txIndex; i++ {
		tx := block.Block.Data.Txs[i]
		_, _ = app.ProcessTx(tx)
	}
	result, tracer, err := app.TraceProcessTx(block.Block.Data.Txs[txIndex], config)

	switch tracer := tracer.(type) {
	case *vm.StructLogger:
		return &ethapi.ExecutionResult{
			Failed:      err == nil,
			ReturnValue: fmt.Sprintf("%x", result),
			StructLogs:  ethapi.FormatLogs(tracer.StructLogs()),
		}, nil
	case *tracers.Tracer:
		return tracer.GetResult()
	default:
		return nil, errors.New(fmt.Sprintf("bad tracer type %T", tracer))
	}
}
