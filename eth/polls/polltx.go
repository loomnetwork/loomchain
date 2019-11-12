// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	"github.com/pkg/errors"
)

type EthTxPoll struct {
	startBlock    uint64
	lastBlockRead uint64
	evmAuxStore   *evmaux.EvmAuxStore
	blockStore    store.BlockStore
	maxBlockRange uint64
}

func NewEthTxPoll(height uint64, evmAuxStore *evmaux.EvmAuxStore, blockStore store.BlockStore, maxBlockRange uint64) *EthTxPoll {
	p := &EthTxPoll{
		startBlock:    height,
		lastBlockRead: height,
		evmAuxStore:   evmAuxStore,
		blockStore:    blockStore,
		maxBlockRange: maxBlockRange,
	}
	return p
}

func (p *EthTxPoll) Poll(
	state loomchain.ReadOnlyState, id string, readReceipt loomchain.ReadReceiptHandler,
) (EthPoll, interface{}, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	toBlock := uint64(state.Block().Height)
	if toBlock-p.lastBlockRead > p.maxBlockRange {
		toBlock = p.lastBlockRead + p.maxBlockRange
	}
	lastBlock, results, err := getTxHashes(p.lastBlockRead, toBlock, readReceipt, p.evmAuxStore)
	if err != nil {
		return p, nil, nil
	}
	p.lastBlockRead = lastBlock
	return p, eth.EncBytesArray(results), nil
}

// AllLogs pull txs from last N blocks limited by p.maxBlockRange
func (p *EthTxPoll) AllLogs(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (interface{}, error) {
	toBlock := uint64(state.Block().Height)
	startBlock := p.startBlock
	if toBlock-startBlock > p.maxBlockRange {
		startBlock = toBlock - p.maxBlockRange
	}
	_, results, err := getTxHashes(startBlock, toBlock, readReceipts, p.evmAuxStore)
	return eth.EncBytesArray(results), err
}

func getTxHashes(lastBlockRead, toBlock uint64,
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *evmaux.EvmAuxStore) (uint64, [][]byte, error) {
	var txHashes [][]byte
	for height := lastBlockRead + 1; height < toBlock; height++ {
		txHashList, err := evmAuxStore.GetTxHashList(height)

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

func (p *EthTxPoll) LegacyPoll(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (EthPoll, []byte, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	toBlock := uint64(state.Block().Height)
	if toBlock-p.lastBlockRead > p.maxBlockRange {
		toBlock = p.lastBlockRead + p.maxBlockRange
	}
	var txHashes [][]byte
	for height := p.lastBlockRead + 1; height <= toBlock; height++ {
		txHashList, err := p.evmAuxStore.GetTxHashList(height)
		if err != nil {
			return p, nil, errors.Wrapf(err, "reading tx hash at heght %d", height)
		}
		if len(txHashList) > 0 {
			txHashes = append(txHashes, txHashList...)
		}
	}
	p.lastBlockRead = uint64(state.Block().Height)

	blocksMsg := types.EthFilterEnvelope_EthTxHashList{
		EthTxHashList: &types.EthTxHashList{EthTxHash: txHashes},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return p, r, err
}
