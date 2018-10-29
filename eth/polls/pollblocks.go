// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/tendermint/tendermint/rpc/core"
)

type EthBlockPoll struct {
	startBlock    uint64
	lastBlockRead uint64
}

func NewEthBlockPoll(height uint64) *EthBlockPoll {
	p := &EthBlockPoll{
		startBlock:    height,
		lastBlockRead: height,
	}

	return p
}

func (p *EthBlockPoll) Poll(state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, interface{}, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	lastBlock, results, err := getBlockHashes(state, p.lastBlockRead)
	if err != nil {
		return p, nil, nil
	}
	p.lastBlockRead = lastBlock
	return p, eth.EncBytesArray(results), err
}

func (p *EthBlockPoll) AllLogs(state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (interface{}, error) {
	_, results, err := getBlockHashes(state, p.startBlock)
	return eth.EncBytesArray(results), err
}

func getBlockHashes(state loomchain.ReadOnlyState, start uint64) (uint64, [][]byte, error) {
	result, err := core.BlockchainInfo(int64(start+1), state.Block().Height)
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

func (p *EthBlockPoll) DepreciatedPoll(state loomchain.ReadOnlyState, id string, readReceipts loomchain.ReadReceiptHandler) (EthPoll, []byte, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	result, err := core.BlockchainInfo(int64(p.lastBlockRead+1), state.Block().Height)
	if err != nil {
		return p, nil, err
	}

	var blockHashes [][]byte
	lastBlock := p.lastBlockRead
	for _, meta := range result.BlockMetas {
		if len(meta.BlockID.Hash) > 0 {
			blockHashes = append(blockHashes, meta.BlockID.Hash)
			if lastBlock < uint64(meta.Header.Height) {
				lastBlock = uint64(meta.Header.Height)
			}
		}
	}
	p.lastBlockRead = lastBlock

	blocksMsg := types.EthFilterEnvelope_EthBlockHashList{
		&types.EthBlockHashList{EthBlockHash: blockHashes},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return p, r, err
}
