package debug

import (
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

func TraceTransaction(app loomchain.InMemoryApp, blockstore store.BlockStore, blockNumber int64, txIndex uint64, config JsonTraceConfig) (interface{}, error) {
	block, err := blockstore.GetBlockByHeight(&blockNumber)
	if err != nil {
		return nil, errors.Wrapf(err, "getting block information at height %v", blockNumber)
	}
	for i := uint64(0); i < txIndex; i++ {
		tx := block.Block.Data.Txs[i]
		_, _ = app.ProcessTx(tx, false)
	}
	_, _ = app.ProcessTx(block.Block.Data.Txs[txIndex], false)
	return nil, errors.New("not implemented")
}
