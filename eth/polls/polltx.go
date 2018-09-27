// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	`github.com/loomnetwork/loomchain/receipts`
	`github.com/pkg/errors`
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

	var txHashes [][]byte
	for height := p.lastBlock + 1; height < uint64(state.Block().Height); height++ {
		txHash, err := readReceipts.GetTxHash(height)
		if err != nil {
			return p, nil, errors.Wrapf(err, "reading tx hash at heght %d", height)
		}
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
