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
	blockNumber, txIndex int64,
	config eth.TraceConfig,
) (interface{}, error) {
	if err := app.RunUpTo(blockNumber, txIndex); err != nil {
		return nil, err
	}

	block, err := blockstore.GetBlockByHeight(&blockNumber)
	if err != nil {
		return nil, errors.Wrapf(err, "getting block information at height %v", blockNumber)
	}

	result, tracer, err := app.TraceProcessTx(block.Block.Data.Txs[txIndex], config)

	switch tracer := tracer.(type) {
	case *vm.StructLogger:
		return &ethapi.ExecutionResult{
			Gas:         5,
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
