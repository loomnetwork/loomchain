// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	`github.com/loomnetwork/loomchain/receipts`
	"github.com/loomnetwork/loomchain/store"
)

type EthTxPoll struct {
	lastBlock uint64
}

func NewEthTxPoll(height uint64) *EthTxPoll {
	p := &EthTxPoll{
		lastBlock: height,
	}
	return p
}

func (p EthTxPoll) Poll(state loomchain.ReadOnlyState, id string, readReceipts receipts.ReadReceiptHandler) (EthPoll, []byte, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	txHashState := store.PrefixKVReader(utils.TxHashPrefix, state)
	var txHashes [][]byte
	for height := p.lastBlock + 1; height < uint64(state.Block().Height); height++ {
		heightB := utils.BlockHeightToBytes(height)
		txHash := txHashState.Get(heightB)
		if len(txHash) > 0 {
			txHashes = append(txHashes, txHash)
		}
	}
	p.lastBlock = uint64(state.Block().Height)

	blocksMsg := types.EthFilterEnvelope_EthTxHashList{
		&types.EthTxHashList{EthTxHash: txHashes},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return p, r, err
}
