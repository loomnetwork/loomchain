// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/tendermint/tendermint/rpc/core"
)

type EthBlockPoll struct {
	lastBlock uint64
	rpcAddr   string
}

func NewEthBlockPoll(height uint64, rpcAddr string) *EthBlockPoll {
	p := &EthBlockPoll{
		lastBlock: height,
		rpcAddr:   rpcAddr,
	}

	return p
}

func (p EthBlockPoll) Poll(state loomchain.ReadOnlyState, id string) (EthPoll, []byte, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	result, err := core.BlockchainInfo(int64(p.lastBlock+1), state.Block().Height)
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
