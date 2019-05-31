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

func (p *EthTxPoll) Poll(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string, readReceipt loomchain.ReadReceiptHandler) (EthPoll, interface{}, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	lastBlock, results, err := getTxHashes(state, p.lastBlockRead, readReceipt)
	if err != nil {
		return p, nil, nil
	}
	p.lastBlockRead = lastBlock
	return p, eth.EncBytesArray(results), nil
}

func (p *EthTxPoll) AllLogs(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (interface{}, error) {
	_, results, err := getTxHashes(state, p.startBlock, readReceipts)
	return eth.EncBytesArray(results), err
}

func getTxHashes(state loomchain.ReadOnlyState, lastBlockRead uint64, readReceipts loomchain.ReadReceiptHandler) (uint64, [][]byte, error) {
	var txHashes [][]byte
	for height := lastBlockRead + 1; height < uint64(state.Block().Height); height++ {
		var txHashList [][]byte
		var err error
		if state.FeatureEnabled(loomchain.ReceiptDBFeature, false) {
			txHashList, err = readReceipts.GetTxHashList(height)
		} else {
			txHashList, err = common.GetTxHashList(state, height)
		}

		if err != nil {
			return lastBlockRead, nil, errors.Wrapf(err, "reading tx hashes at height %d", height)
		}
		if len(txHashList) > 0 {
			txHashes = append(txHashes, txHashList...)
		}
		lastBlockRead = height
	}
	return lastBlockRead, txHashes, nil
}

func (p *EthTxPoll) LegacyPoll(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, []byte, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	var txHashes [][]byte
	for height := p.lastBlockRead + 1; height < uint64(state.Block().Height); height++ {
		var txHashList [][]byte
		var err error
		if state.FeatureEnabled(loomchain.ReceiptDBFeature, false) {
			txHashList, err = readReceipts.GetTxHashList(height)
		} else {
			txHashList, err = common.GetTxHashList(state, height)
		}
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
