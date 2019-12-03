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
	startBlock := p.lastBlock + 1
	toBlock := uint64(state.Block().Height)

	if startBlock > toBlock {
		return p, nil, nil
	}

	if toBlock-startBlock > p.maxBlockRange {
		toBlock = startBlock + p.maxBlockRange
	}

	lastBlockRead, results, err := getBlockHashes(p.blockStore, startBlock, toBlock)
	if err != nil {
		return p, nil, nil
	}

	p.lastBlock = lastBlockRead
	return p, eth.EncBytesArray(results), err
}

// AllLogs returns logs from the latest maxBlockRange blocks.
func (p *EthBlockPoll) AllLogs(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (interface{}, error) {
	// NOTE: startBlock & lastBlock should never be modified in this function
	toBlock := uint64(state.Block().Height)
	startBlock := p.startBlock
	if toBlock-startBlock > p.maxBlockRange {
		startBlock = toBlock - p.maxBlockRange
	}
	_, results, err := getBlockHashes(p.blockStore, startBlock, toBlock)
	return eth.EncBytesArray(results), err
}

func getBlockHashes(
	blockStore store.BlockStore, fromBlock, toBlock uint64,
) (uint64, [][]byte, error) {
	result, err := blockStore.GetBlockRangeByHeight(int64(fromBlock), int64(toBlock))
	if err != nil {
		return 0, nil, err
	}

	lastBlockRead := int64(fromBlock)
	var blockHashes [][]byte
	for _, meta := range result.BlockMetas {
		if len(meta.BlockID.Hash) > 0 {
			blockHashes = append(blockHashes, meta.BlockID.Hash)
			if lastBlockRead < meta.Header.Height {
				lastBlockRead = meta.Header.Height
			}
		}
	}
	return uint64(lastBlockRead), blockHashes, nil
}

func (p *EthBlockPoll) LegacyPoll(
	state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler,
) (EthPoll, []byte, error) {
	startBlock := p.lastBlock + 1
	toBlock := uint64(state.Block().Height)

	if startBlock > toBlock {
		return p, nil, nil
	}

	if toBlock-startBlock > p.maxBlockRange {
		toBlock = startBlock + p.maxBlockRange
	}

	lastBlockRead, blockHashes, err := getBlockHashes(p.blockStore, startBlock, toBlock)
	if err != nil {
		return p, nil, nil
	}

	p.lastBlock = lastBlockRead
	blocksMsg := types.EthFilterEnvelope_EthBlockHashList{
		EthBlockHashList: &types.EthBlockHashList{EthBlockHash: blockHashes},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return p, r, err
}
