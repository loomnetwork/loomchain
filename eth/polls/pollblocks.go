// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
)

type EthBlockPoll struct {
	startBlock    uint64
	lastBlock     uint64
	evmAuxStore   *evmaux.EvmAuxStore
	blockStore    store.BlockStore
	maxBlockRange uint64
}

func NewEthBlockPoll(
	height uint64, evmAuxStore *evmaux.EvmAuxStore, blockStore store.BlockStore, maxBlockRange uint64,
) *EthBlockPoll {
	p := &EthBlockPoll{
		startBlock:    height,
		lastBlock:     height,
		evmAuxStore:   evmAuxStore,
		blockStore:    blockStore,
		maxBlockRange: maxBlockRange,
	}
	return p
}

func (p *EthBlockPoll) Poll(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (EthPoll, interface{}, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	toBlock := uint64(state.Block().Height)
	if toBlock-p.lastBlock > p.maxBlockRange {
		toBlock = p.lastBlock + p.maxBlockRange
	}
	lastBlock, results, err := getBlockHashes(p.blockStore, toBlock, p.lastBlock)
	if err != nil {
		return p, nil, nil
	}
	p.lastBlock = lastBlock
	return p, eth.EncBytesArray(results), err
}

// AllLogs pull logs from last N blocks limited by p.maxBlockRange
func (p *EthBlockPoll) AllLogs(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (interface{}, error) {
	// Only pull logs from latest height - maxBlockRange
	toBlock := uint64(state.Block().Height)
	startBlock := p.startBlock
	if toBlock-startBlock > p.maxBlockRange {
		startBlock = toBlock - p.maxBlockRange
	}
	_, results, err := getBlockHashes(p.blockStore, toBlock, startBlock)
	return eth.EncBytesArray(results), err
}

func getBlockHashes(
	blockStore store.BlockStore, toBlock, lastBlockRead uint64,
) (uint64, [][]byte, error) {
	result, err := blockStore.GetBlockRangeByHeight(int64(lastBlockRead+1), int64(toBlock))
	if err != nil {
		return lastBlockRead, nil, err
	}

	var blockHashes [][]byte
	for _, meta := range result.BlockMetas {
		if len(meta.BlockID.Hash) > 0 {
			blockHashes = append(blockHashes, meta.BlockID.Hash)
			if lastBlockRead < uint64(meta.Header.Height) {
				lastBlockRead = uint64(meta.Header.Height)
			}
		}
	}
	return lastBlockRead, blockHashes, nil
}

func (p *EthBlockPoll) LegacyPoll(state loomchain.ReadOnlyState, id string,
	readReceipts loomchain.ReadReceiptHandler) (EthPoll, []byte, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	toBlock := uint64(state.Block().Height)
	if toBlock-p.lastBlock > p.maxBlockRange {
		toBlock = p.lastBlock + p.maxBlockRange
	}
	result, err := p.blockStore.GetBlockRangeByHeight(int64(p.lastBlock+1), int64(toBlock))
	if err != nil {
		return p, nil, err
	}

	var blockHashes [][]byte
	lastBlock := p.lastBlock
	for _, meta := range result.BlockMetas {
		if len(meta.BlockID.Hash) > 0 {
			blockHashes = append(blockHashes, meta.BlockID.Hash)
			if lastBlock < uint64(meta.Header.Height) {
				lastBlock = uint64(meta.Header.Height)
			}
		}
	}

	p.lastBlock = lastBlock
	blocksMsg := types.EthFilterEnvelope_EthBlockHashList{
		EthBlockHashList: &types.EthBlockHashList{EthBlockHash: blockHashes},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return p, r, err
}
