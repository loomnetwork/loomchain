// +build evm

package debug

import (
	"bytes"
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

	tracer, err := CreateTracer(config)
	if err != nil {
		return nil, err
	}
	if err := app.SetTracer(tracer, false); err != nil {
		return nil, err
	}

	txResult, err := blockstore.GetTxResult(block.Block.Data.Txs[txIndex].Hash())
	if err != nil {
		return nil, err
	}

	if err := app.SetTracer(tracer, false); err != nil {
		return nil, err
	}
	result := app.DeliverTx(block.Block.Data.Txs[txIndex])
	match, err := resultsMatch(txResult.TxResult, result)
	if !match {
		return nil, err
	}

	switch tracer := tracer.(type) {
	case *vm.StructLogger:
		return &ethapi.ExecutionResult{
			Gas:         0,
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
			return errors.Wrapf(err, "getting block information at height %v", h)
		}

		_ = app.BeginBlock(requestBeginBlock(*resultBlock))

		if h != height {
			for i := 0; i < len(resultBlock.Block.Data.Txs); i++ {
				_ = app.DeliverTx(resultBlock.Block.Data.Txs[i])
			}
		} else {
			for i := 0; i < len(resultBlock.Block.Data.Txs); i++ {
				if i != int(index) {
					_ = app.DeliverTx(resultBlock.Block.Data.Txs[i])
				} else {
					return nil
				}
			}
		}

		_ = app.EndBlock(requestEndBlock(h))
		_ = app.Commit()
	}
	return errors.Errorf("cannot find transaction at height %d index %d", height, index)
}

func resultsMatch(expected, actual abci.ResponseDeliverTx) (bool, error) {
	if expected.Code != actual.Code {
		return false, errors.Errorf("transaction result codes do not match, expected %v got $v", expected.Code, actual.Code)
	}
	if 0 == bytes.Compare(expected.Data, actual.Data) {
		return false, errors.Errorf("transaction result data does not match, expected %v got $v", expected.Data, actual.Data)
	}
	if expected.Log != actual.Log {
		return false, errors.Errorf("transaction logs do not match, expected %v got $v", expected.Log, actual.Log)
	}
	if expected.Info != actual.Info {
		return false, errors.Errorf("transaction info does not match, expected %v got $v", expected.Info, actual.Info)
	}
	return true, nil
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

func CreateTracer(traceCfg eth.TraceConfig) (vm.Tracer, error) {
	if traceCfg.Tracer == nil || len(*traceCfg.Tracer) == 0 {
		return vm.NewStructLogger(traceCfg.LogConfig), nil
	}
	tracer, err := tracers.New(*traceCfg.Tracer)
	if err != nil {
		return nil, err
	}
	return tracer, nil
}
