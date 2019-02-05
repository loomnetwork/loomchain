// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
)

type EthTxPoll struct {
	startBlock    uint64
	lastBlockRead uint64
}

func NewEthTxPoll(height uint64) *EthTxPoll {
	p := &EthTxPoll{
		startBlock:    height,
		lastBlockRead: height,
	}
	return p
}

func (p *EthTxPoll) Poll(
	blockStore store.BlockStore,
	state loomchain.ReadOnlyState,
	id string,
	_ loomchain.ReadReceiptHandler,
) (EthPoll, interface{}, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	lastBlock, results, err := getTxHashes(state, p.lastBlockRead)
	if err != nil {
		return p, nil, nil
	}
	p.lastBlockRead = lastBlock
	return p, eth.EncBytesArray(results), err
}

func (p *EthTxPoll) AllLogs(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (interface{}, error) {
	_, results, err := getTxHashes(state, p.startBlock)
	return eth.EncBytesArray(results), err
}

func getTxHashes(state loomchain.ReadOnlyState, start uint64) (uint64, [][]byte, error) {
	var txHashes [][]byte
	for height := start + 1; height < uint64(state.Block().Height); height++ {
		txHashList, err := common.GetTxHashList(state, height)
		if err != nil {
			return start, nil, errors.Wrapf(err, "reading tx hash at heght %d", height)
		}
		if len(txHashList) > 0 {
			txHashes = append(txHashes, txHashList...)
		}
	}
	start = uint64(state.Block().Height)
	return start, txHashes, nil
}

func (p *EthTxPoll) DepreciatedPoll(state loomchain.ReadOnlyState, id string, _ loomchain.ReadReceiptHandler) (EthPoll, []byte, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	var txHashes [][]byte
	for height := p.lastBlockRead + 1; height < uint64(state.Block().Height); height++ {
		txHashList, err := common.GetTxHashList(state, height)
		if err != nil {
			return p, nil, errors.Wrapf(err, "reading tx hash at heght %d", height)
		}
		if len(txHashList) > 0 {
			txHashes = append(txHashes, txHashList...)
		}
	}
	p.lastBlockRead = uint64(state.Block().Height)

	blocksMsg := types.EthFilterEnvelope_EthTxHashList{
		&types.EthTxHashList{EthTxHash: txHashes},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return p, r, err
}
