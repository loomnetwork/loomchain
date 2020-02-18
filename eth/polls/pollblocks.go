// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"

	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
)

type EthBlockPoll struct {
	startBlock  uint64
	lastBlock   uint64
	evmAuxStore *evmaux.EvmAuxStore
	blockStore  store.BlockStore
}

func NewEthBlockPoll(height uint64, evmAuxStore *evmaux.EvmAuxStore, blockStore store.BlockStore) *EthBlockPoll {
	p := &EthBlockPoll{
		startBlock:  height,
		lastBlock:   height,
		evmAuxStore: evmAuxStore,
		blockStore:  blockStore,
	}

	return p
}

func (p *EthBlockPoll) Poll(
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	_ func(loomchain.State, loom.Address) (loom.Address, error),

) (EthPoll, interface{}, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	lastBlock, results, err := getBlockHashes(p.blockStore, state, p.lastBlock)
	if err != nil {
		return p, nil, nil
	}
	p.lastBlock = lastBlock
	return p, eth.EncBytesArray(results), err
}

func (p *EthBlockPoll) AllLogs(
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	_ func(loomchain.State, loom.Address) (loom.Address, error),
) (interface{}, error) {
	_, results, err := getBlockHashes(p.blockStore, state, p.startBlock)
	return eth.EncBytesArray(results), err
}

func getBlockHashes(
	blockStore store.BlockStore, state loomchain.State, lastBlockRead uint64,
) (uint64, [][]byte, error) {
	result, err := blockStore.GetBlockRangeByHeight(int64(lastBlockRead+1), state.Block().Height)
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

func (p *EthBlockPoll) LegacyPoll(
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	_ func(loomchain.State, loom.Address) (loom.Address, error),
) (EthPoll, []byte, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	result, err := p.blockStore.GetBlockRangeByHeight(int64(p.lastBlock+1), state.Block().Height)
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
