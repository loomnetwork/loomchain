// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
)

type EthBlockPoll struct {
	startBlock uint64
	lastBlock  uint64
}

func NewEthBlockPoll(height uint64) *EthBlockPoll {
	p := &EthBlockPoll{
		startBlock: height,
		lastBlock:  height,
	}

	return p
}

func (p *EthBlockPoll) Poll(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, interface{}, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	lastBlock, results, err := getBlockHashes(blockStore, state, p.lastBlock)
	if err != nil {
		return p, nil, nil
	}
	p.lastBlock = lastBlock
	return p, eth.EncBytesArray(results), err
}

func (p *EthBlockPoll) AllLogs(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (interface{}, error) {
	_, results, err := getBlockHashes(blockStore, state, p.startBlock)
	return eth.EncBytesArray(results), err
}

func getBlockHashes(blockStore store.BlockStore, state loomchain.ReadOnlyState, start uint64) (uint64, [][]byte, error) {
	result, err := blockStore.GetBlockRangeByHeight(int64(start+1), state.Block().Height)
	if err != nil {
		return start, nil, err
	}

	var blockHashes [][]byte
	for _, meta := range result.BlockMetas {
		if len(meta.BlockID.Hash) > 0 {
			blockHashes = append(blockHashes, meta.BlockID.Hash)
			if start < uint64(meta.Header.Height) {
				start = uint64(meta.Header.Height)
			}
		}
	}
	return start, blockHashes, nil
}

func (p *EthBlockPoll) DepreciatedPoll(blockStore store.BlockStore, state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, []byte, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	result, err := blockStore.GetBlockRangeByHeight(int64(p.lastBlock+1), state.Block().Height)
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
		&types.EthBlockHashList{EthBlockHash: blockHashes},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return p, r, err
}
