// +build evm

package debug

import (
	"github.com/ethereum/go-ethereum/eth"
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

	result, err := app.TraceProcessTx(block.Block.Data.Txs[txIndex], config)
	if err != nil {
		return nil, err
	}
	return result, errors.New("not implemented")
}
