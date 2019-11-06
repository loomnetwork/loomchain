// +build evm

package debug

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethapi"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

func TraceTransaction(
	app loomchain.Application,
	blockstore store.BlockStore,
	startBlockNumber, targetBlockNumber, txIndex int64,
	config eth.TraceConfig,
) (trace interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("loomchain panicked %v", r)
		}
	}()

	if err := runUpTo(&app, blockstore, startBlockNumber, targetBlockNumber, txIndex); err != nil {
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
	if err := app.SetTracer(tracer, false); err != nil {
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

func runUpTo(app *loomchain.Application, blockstore store.BlockStore, startHeight, height, index int64) error {
	for h := startHeight; h <= height; h++ {
		resultBlock, err := blockstore.GetBlockByHeight(&h)
		if err != nil {
			return err
		}
		_ = app.BeginBlock(requestBeginBlock(*resultBlock))

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

		_ = app.EndBlock(requestEndBlock(h))
		_ = app.Commit()
	}
	return errors.Errorf("cannot find transaction at height %d index %d", height, index)
}

func requestBeginBlock(resultBlock ctypes.ResultBlock) abci.RequestBeginBlock {
	return abci.RequestBeginBlock{
		Header: ttypes.TM2PB.Header(&resultBlock.BlockMeta.Header),
		Hash:   resultBlock.BlockMeta.BlockID.Hash,
	}
}

func requestEndBlock(height int64) abci.RequestEndBlock {
	return abci.RequestEndBlock{
		Height: height,
	}
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
