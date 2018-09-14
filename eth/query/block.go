// +build evm

package query

import (
	"bytes"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/store"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

var (
	searchBlockSize = uint64(100)
)

func GetBlockByNumber(state loomchain.ReadOnlyState, height uint64, full bool) ([]byte, error) {
	params := map[string]interface{}{}
	params["heightPtr"] = &height
	var blockresult *ctypes.ResultBlock
	iHeight := int64(height)
	blockresult, err := core.Block(&iHeight)
	if err != nil {
		return nil, err
	}
	blockinfo := types.EthBlockInfo{
		Hash:       blockresult.BlockMeta.BlockID.Hash,
		ParentHash: blockresult.Block.Header.LastBlockID.Hash,

		Timestamp: int64(blockresult.Block.Header.Time.Unix()),
	}
	if uint64(state.Block().Height) == height {
		blockinfo.Number = 0
	} else {
		blockinfo.Number = int64(height)
	}

	txHashState := store.PrefixKVReader(utils.TxHashPrefix, state)
	txHash := txHashState.Get(utils.BlockHeightToBytes(height))
	if len(txHash) > 0 {
		receiptState := store.PrefixKVReader(utils.ReceiptPrefix, state)
		txReceiptProto := receiptState.Get(txHash)
		txReceipt := types.EvmTxReceipt{}
		if err := proto.Unmarshal(txReceiptProto, &txReceipt); err != nil {
			return nil, err
		}
		blockinfo.LogsBloom = txReceipt.LogsBloom
		if full {
			blockinfo.Transactions = append(blockinfo.Transactions, txReceiptProto)
		} else {
			blockinfo.Transactions = append(blockinfo.Transactions, txHash)
		}
	}

	return proto.Marshal(&blockinfo)
}

func GetBlockByHash(state loomchain.ReadOnlyState, hash []byte, full bool) ([]byte, error) {
	start := uint64(state.Block().Height)
	var end uint64
	if uint64(start) > searchBlockSize {
		end = uint64(start) - searchBlockSize
	} else {
		end = 1
	}

	for start > 0 {
		var info *ctypes.ResultBlockchainInfo
		info, err := core.BlockchainInfo(int64(end), int64(start))
		if err != nil {
			return nil, err
		}

		if err != nil {
			return nil, err
		}
		for i := int(len(info.BlockMetas) - 1); i >= 0; i-- {
			if 0 == bytes.Compare(hash, info.BlockMetas[i].BlockID.Hash) {
				return GetBlockByNumber(state, uint64(int(end)+i), full)
			}
		}

		if end == 1 {
			return nil, fmt.Errorf("can't find block to match hash")
		}

		start = end
		if uint64(start) > searchBlockSize {
			end = uint64(start) - searchBlockSize
		} else {
			end = 1
		}
	}
	return nil, fmt.Errorf("can't find block to match hash")
}
