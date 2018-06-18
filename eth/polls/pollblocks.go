// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/rpc/lib/client"
)

type EthBlockPoll struct {
	lastBlock uint64
}

func NewEthBlockPoll(height uint64) *EthBlockPoll {
	p := &EthBlockPoll{
		lastBlock: height,
	}
	return p
}

func (p EthBlockPoll) Poll(state loomchain.ReadOnlyState, id string) (EthPoll, []byte, error) {
	if p.lastBlock+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	params := map[string]interface{}{}
	params["minHeight"] = int64(p.lastBlock + 1)
	params["maxHeight"] = state.Block().Height
	var result ctypes.ResultBlockchainInfo
	rclient := rpcclient.NewJSONRPCClient("tcp://0.0.0.0:46657")
	_, err := rclient.Call("blockchain", params, &result)
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

	r, err := proto.Marshal(&types.EthBlockHashList{blockHashes})
	return p, r, err
}
