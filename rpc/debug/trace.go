// +build evm

package debug

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/ethereum/go-ethereum/eth/ethapi"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/loomnetwork/loomchain/replay"
	"github.com/loomnetwork/loomchain/store"
)

func TraceTransaction(
	app replay.ReplayApplication,
	blockstore store.BlockStore,
	startBlockNumber, targetBlockNumber, txIndex int64,
	config eth.TraceConfig,
) (interface{}, error) {
	if err := runUpTo(app, blockstore, startBlockNumber, targetBlockNumber, txIndex); err != nil {
		return nil, err
	}

	block, err := blockstore.GetBlockByHeight(&targetBlockNumber)
	if err != nil {
		return nil, errors.Wrapf(err, "getting block information at height %v", targetBlockNumber)
	}
	tracer, err := createTracer(config)
	if err != nil {
		return nil, err
	}
	if err := app.SetTracer(tracer); err != nil {
		return nil, err
	}
	result := app.DeliverTx(block.Block.Data.Txs[txIndex])

	switch tracer := tracer.(type) {
	case *vm.StructLogger:
		return &ethapi.ExecutionResult{
			Gas:         5,
			Failed:      result.Code != abci.CodeTypeOK,
			ReturnValue: fmt.Sprintf("%x", result),
			StructLogs:  ethapi.FormatLogs(tracer.StructLogs()),
		}, nil
	case *tracers.Tracer:
		return tracer.GetResult()
	default:
		return nil, errors.New(fmt.Sprintf("bad tracer type %T", tracer))
	}
}

func runUpTo(app abci.Application, blockstore store.BlockStore, startHeight, height, index int64) error {
	for h := startHeight; h <= height; h++ {
		_ = app.BeginBlock(abci.RequestBeginBlock{})

		block, err := blockstore.GetBlockByHeight(&h)
		if err != nil {
			return errors.Wrapf(err, "getting block information at height %v", h)
		}
		for i := 0; i < len(block.Block.Data.Txs); i++ {
			if h != height || i != int(index) {
				_ = app.DeliverTx(block.Block.Data.Txs[i])
			} else {
				return nil
			}
		}

		_ = app.EndBlock(abci.RequestEndBlock{})
		_ = app.Commit()
	}
	return errors.Errorf("cannot find transaction at height %d index %d", height, index)
}

func createTracer(traceCfg eth.TraceConfig) (vm.Tracer, error) {
	if traceCfg.Tracer == nil || len(*traceCfg.Tracer) == 0 {
		return vm.NewStructLogger(traceCfg.LogConfig), nil
	}
	tracer, err := tracers.New(*traceCfg.Tracer)
	if err != nil {
		return nil, err
	}
	return tracer, nil
}
