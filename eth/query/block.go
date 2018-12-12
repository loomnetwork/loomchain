// +build evm

package query

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

var (
	searchBlockSize = uint64(20)
)

func GetBlockByNumber(state loomchain.ReadOnlyState, height int64, full bool, readReceipts loomchain.ReadReceiptHandler) (resp eth.JsonBlockObject, err error) {
	// todo make information about pending block avaliable
	if height > state.Block().Height {
		return resp, errors.New("get block information for pending blocks not implemented yet")
	}

	var blockResult *ctypes.ResultBlock
	blockResult, err = core.Block(&height)
	if err != nil {
		return resp, err
	}

	blockinfo := eth.JsonBlockObject{
		ParentHash:   eth.EncBytes(blockResult.Block.Header.LastBlockID.Hash),
		Timestamp:    eth.EncInt(int64(blockResult.Block.Header.Time.Unix())),
		GasLimit:     eth.EncInt(0),
		GasUsed:      eth.EncInt(0),
		Size:         eth.EncInt(0),
		Transactions: nil,
	}

	// These three fields are null for pending blocks.
	blockinfo.Hash = eth.EncBytes(blockResult.BlockMeta.BlockID.Hash)
	blockinfo.Number = eth.EncInt(height)
	blockinfo.LogsBloom = eth.EncBytes(common.GetBloomFilter(state, uint64(height)))

	txHashList, err := common.GetTxHashList(state, uint64(height))
	if err != nil {
		return resp, errors.Wrapf(err, "get tx hash list at height %v", height)
	}
	for _, hash := range txHashList {
		if full {
			txObj, err := GetTxByHash(state, hash, readReceipts)
			if err != nil {
				return resp, errors.Wrapf(err, "txObj for hash %v", hash)
			}
			blockinfo.Transactions = append(blockinfo.Transactions, txObj)
		} else {
			blockinfo.Transactions = append(blockinfo.Transactions, eth.EncBytes(hash))
		}
	}
	return blockinfo, nil
}

func GetNumEvmTxBlock(state loomchain.ReadOnlyState, height int64) (uint64, error) {
	// todo make information about pending block available.
	// Should be able to get transaction count from receipt object.
	if height > state.Block().Height {
		return 0, errors.New("get number of transcations for pending blocks, not implemented yet")
	}

	var blockResults *ctypes.ResultBlockResults
	blockResults, err := core.BlockResults(&height)
	if err != nil {
		return 0, errors.Wrapf(err, "results for block %v", height)
	}

	numEvmTx := uint64(0)
	for _, deliverTx := range blockResults.Results.DeliverTx {
		if deliverTx.Info == utils.DeployEvm || deliverTx.Info == utils.CallEVM {
			numEvmTx++
		}
	}
	return numEvmTx, nil
}

// todo find better method of doing this. Maybe use a blockhash index.
func GetBlockHeightFromHash(state loomchain.ReadOnlyState, hash []byte) (int64, error) {
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
			return 0, err
		}

		if err != nil {
			return 0, err
		}
		for i := int(len(info.BlockMetas) - 1); i >= 0; i-- {
			if 0 == bytes.Compare(hash, info.BlockMetas[i].BlockID.Hash) {
				return info.BlockMetas[i].Header.Height, nil //    int64(int(end) + i), nil
			}
		}

		if end == 1 {
			return 0, fmt.Errorf("can't find block to match hash")
		}

		start = end
		if uint64(start) > searchBlockSize {
			end = uint64(start) - searchBlockSize
		} else {
			end = 1
		}
	}
	return 0, fmt.Errorf("can't find block to match hash")
}

func DeprecatedGetBlockByNumber(state loomchain.ReadOnlyState, height int64, full bool, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	var blockresult *ctypes.ResultBlock
	iHeight := height
	blockresult, err := core.Block(&iHeight)
	if err != nil {
		return nil, err
	}
	blockinfo := types.EthBlockInfo{
		Hash:       blockresult.BlockMeta.BlockID.Hash,
		ParentHash: blockresult.Block.Header.LastBlockID.Hash,

		Timestamp: int64(blockresult.Block.Header.Time.Unix()),
	}
	if state.Block().Height == height {
		blockinfo.Number = 0
	} else {
		blockinfo.Number = height
	}

	bloomFilter := common.GetBloomFilter(state, uint64(height))
	blockinfo.LogsBloom = bloomFilter

	txHashList, err := common.GetTxHashList(state, uint64(height))
	if err != nil {
		return nil, errors.Wrap(err, "getting tx hash")
	}
	if full {
		for _, txHash := range txHashList {
			txObj, err := DeprecatedGetTxByHash(state, txHash, readReceipts)
			if err != nil {
				return nil, errors.Wrap(err, "marshall tx object")
			}
			blockinfo.Transactions = append(blockinfo.Transactions, txObj)
		}
	} else {
		blockinfo.Transactions = txHashList
	}

	return proto.Marshal(&blockinfo)
}

func GetPendingBlock(height int64, full bool, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	blockinfo := types.EthBlockInfo{
		Number: int64(height),
	}
	txHashList := readReceipts.GetPendingTxHashList()
	if full {
		for _, txHash := range txHashList {
			txReceipt, err := readReceipts.GetPendingReceipt(txHash)
			if err != nil {
				return nil, errors.Wrap(err, "reading receipt")
			}
			txReceiptProto, err := proto.Marshal(&txReceipt)
			if err != nil {
				return nil, errors.Wrap(err, "marshall receipt")
			}
			blockinfo.Transactions = append(blockinfo.Transactions, txReceiptProto)
		}
	} else {
		blockinfo.Transactions = txHashList
	}

	return proto.Marshal(&blockinfo)
}

func DeprecatedGetBlockByHash(state loomchain.ReadOnlyState, hash []byte, full bool, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
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
				return DeprecatedGetBlockByNumber(state, int64(int(end)+i), full, readReceipts)
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
