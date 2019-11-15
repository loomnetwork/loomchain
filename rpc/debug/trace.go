// +build evm

package debug

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethapi"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/evm"
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

	block, err := runTxsTo(&app, blockstore, startBlockNumber, targetBlockNumber, txIndex, false)
	if err != nil {
		return nil, err
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

func StorageRangeAt(
	app loomchain.Application,
	blockstore store.BlockStore,
	address, begin []byte,
	startBlockNumber, targetBlockNumber, txIndex int64,
	maxResults int,
) (results JsonStorageRangeResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("loomchain panicked %v", r)
		}
	}()

	block, err := runTxsTo(&app, blockstore, startBlockNumber, targetBlockNumber, txIndex, true)
	if err != nil {
		return JsonStorageRangeResult{}, err
	}

	storeState := loomchain.NewStoreState(
		context.Background(),
		app.Store,
		ttypes.TM2PB.Header(&block.Block.Header),
		block.BlockMeta.BlockID.Hash,
		app.GetValidatorSet,
	)
	ethDb := evm.NewLoomEthdb(storeState, nil)
	root := app.Store.Get(util.PrefixKey(evm.VmPrefix, evm.RootKey))

	stateDb, err := state.New(common.BytesToHash(root), state.NewDatabase(ethDb))
	if err != nil {
		return JsonStorageRangeResult{}, err
	}
	st := stateDb.StorageTrie(common.BytesToAddress(address))
	result, err := eth.StorageRangeAt(st, begin, maxResults)
	if err != nil {
		return JsonStorageRangeResult{}, err
	}

	return JsonStorageRangeResult{
		StorageRangeResult: result,
		Complete:           result.NextKey == nil,
	}, nil
}

func runTxsTo(app *loomchain.Application, blockstore store.BlockStore, startHeight, height, index int64, includeEnd bool) (*ctypes.ResultBlock, error) {
	for h := startHeight; h <= height; h++ {
		resultBlock, err := blockstore.GetBlockByHeight(&h)
		if err != nil {
			return nil, errors.Wrapf(err, "getting block information at height %v", h)
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
					if includeEnd {
						_ = app.DeliverTx(resultBlock.Block.Data.Txs[i])
					}
					return resultBlock, nil
				}
			}
		}

		_ = app.EndBlock(requestEndBlock(h))
		_ = app.Commit()
	}
	return nil, errors.Errorf("cannot find transaction at height %d index %d", height, index)
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
