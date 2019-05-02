// +build evm

package query

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
)
const (
	deployId    = uint32(1)
	callId      = uint32(2)
	migrationTx = uint32(3)
)

var (
	searchBlockSize = uint64(20)
)

func GetBlockByNumber(
	blockStore store.BlockStore, state loomchain.ReadOnlyState, height int64, full bool,
) (resp eth.JsonBlockObject, err error) {
	// todo make information about pending block available
	if height > state.Block().Height {
		return resp, errors.New("get block information for pending blocks not implemented yet")
	}

	var blockResult *ctypes.ResultBlock
	blockResult, err = blockStore.GetBlockByHeight(&height)
	if err != nil {
		return resp, err
	}

	var proposalAddress eth.Data

	if blockResult.Block.Header.ProposerAddress != nil {
		proposalAddress = eth.EncBytes(blockResult.Block.Header.ProposerAddress)
	} else {
		proposalAddress = eth.ZeroedData20Bytes
	}

	blockInfo := eth.JsonBlockObject{
		ParentHash:       eth.EncBytes(blockResult.Block.Header.LastBlockID.Hash),
		Timestamp:        eth.EncInt(int64(blockResult.Block.Header.Time.Unix())),
		GasLimit:         eth.EncInt(0),
		GasUsed:          eth.EncInt(0),
		Size:             eth.EncInt(0),
		Transactions:     nil,
		Nonce:            eth.ZeroedData8Bytes,
		Sha3Uncles:       eth.ZeroedData32Bytes,
		TransactionsRoot: eth.ZeroedData32Bytes,
		StateRoot:        eth.ZeroedData32Bytes,
		ReceiptsRoot:     eth.ZeroedData32Bytes,
		Miner:            proposalAddress,
		Difficulty:       eth.ZeroedQuantity,
		TotalDifficulty:  eth.ZeroedQuantity,
		ExtraData:        eth.ZeroedData,
		Uncles:           []eth.Data{},
	}

	// These three fields are null for pending blocks.
	blockInfo.Hash = eth.EncBytes(blockResult.BlockMeta.BlockID.Hash)
	blockInfo.Number = eth.EncInt(height)
	blockInfo.LogsBloom = eth.EncBytes(common.GetBloomFilter(state, uint64(height)))

	for _, tx := range blockResult.Block.Data.Txs {
		if full {
			txResult, err := blockStore.GetTxResult(tx.Hash())
			if err != nil {
				return resp, errors.Wrapf(err, "cant find result for tx, hash %v", tx.Hash())
			}

			txObj, err := GetTxObjectFromTxResult(txResult, blockResult.BlockMeta.BlockID.Hash)
			if err != nil {
				return resp, errors.Wrapf(err, "cant resolve tx, hash %v", tx.Hash())
			}
			blockInfo.Transactions = append(blockInfo.Transactions, txObj)
		} else {
			blockInfo.Transactions = append(blockInfo.Transactions, eth.EncBytes(tx.Hash()))
		}
	}

	if len(blockInfo.Transactions) == 0 {
		blockInfo.Transactions = make([]interface{}, 0)
	}

	return blockInfo, nil
}

func GetTxObjectFromTxResult(txResult *ctypes.ResultTx, blockHash []byte) (eth.JsonTxObject, error) {
	var signedTx auth.SignedTx
	if err := proto.Unmarshal([]byte(txResult.Tx), &signedTx); err != nil {
		return eth.JsonTxObject{}, err
	}

	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return eth.JsonTxObject{}, err
	}

	var txTx loomchain.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &txTx); err != nil {
		return eth.JsonTxObject{}, err
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(txTx.Data, &msg); err != nil {
		return eth.JsonTxObject{}, err
	}

	var input []byte
	switch txTx.Id {
	case deployId:
		{
			var deployTx vm.DeployTx
			if err := proto.Unmarshal(msg.Data, &deployTx);  err != nil {
				return eth.JsonTxObject{}, err
			}
			input = deployTx.Code
		}
	case callId:
		{
			var callTx vm.CallTx
			if err := proto.Unmarshal(msg.Data, &callTx);  err != nil {
				return eth.JsonTxObject{}, err
			}
			input = callTx.Input
		}
	case migrationTx:
		input = msg.Data
	default:
		return eth.JsonTxObject{}, fmt.Errorf("unrecognised tx type %v", txTx.Id)
	}

	return eth.JsonTxObject{
		Nonce:                  eth.EncInt(int64(nonceTx.Sequence)),
		Hash:                   eth.EncBytes(txResult.Hash),
		BlockHash:              eth.EncBytes(blockHash),
		BlockNumber:            eth.EncInt(txResult.Height),
		TransactionIndex:       eth.EncInt(int64(txResult.Index)),
		From:                   eth.EncAddress(msg.From),
		To:                     eth.EncAddress(msg.To),
		Value:                  eth.EncInt(0),
		GasPrice:               eth.EncInt(txResult.TxResult.GasWanted),
		Gas:                    eth.EncInt(txResult.TxResult.GasUsed),
		Input:                  eth.EncBytes(input),
	}, nil
}

func GetNumTxBlock(blockStore store.BlockStore, state loomchain.ReadOnlyState, height int64) (uint64, error) {
	// todo make information about pending block available.
	// Should be able to get transaction count from receipt object.
	if height > state.Block().Height {
		return 0, errors.New("get number of transactions for pending blocks, not implemented yet")
	}

	var blockResults *ctypes.ResultBlockResults
	blockResults, err := blockStore.GetBlockResults(&height)
	if err != nil {
		return 0, errors.Wrapf(err, "results for block %v", height)
	}
	return uint64(len(blockResults.Results.DeliverTx)), nil
}

// todo find better method of doing this. Maybe use a blockhash index.
func GetBlockHeightFromHash(blockStore store.BlockStore, state loomchain.ReadOnlyState, hash []byte) (int64, error) {
	start := uint64(state.Block().Height)
	var end uint64
	if uint64(start) > searchBlockSize {
		end = uint64(start) - searchBlockSize
	} else {
		end = 1
	}

	for start > 0 {
		var info *ctypes.ResultBlockchainInfo
		info, err := blockStore.GetBlockRangeByHeight(int64(end), int64(start))
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

func DeprecatedGetBlockByNumber(
	blockStore store.BlockStore, state loomchain.ReadOnlyState, height int64, full bool,
	readReceipts loomchain.ReadReceiptHandler,
) ([]byte, error) {
	var blockresult *ctypes.ResultBlock
	iHeight := height
	blockresult, err := blockStore.GetBlockByHeight(&iHeight)
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

func DeprecatedGetBlockByHash(
	blockStore store.BlockStore, state loomchain.ReadOnlyState, hash []byte, full bool,
	readReceipts loomchain.ReadReceiptHandler,
) ([]byte, error) {
	start := uint64(state.Block().Height)
	var end uint64
	if uint64(start) > searchBlockSize {
		end = uint64(start) - searchBlockSize
	} else {
		end = 1
	}

	for start > 0 {
		var info *ctypes.ResultBlockchainInfo
		info, err := blockStore.GetBlockRangeByHeight(int64(end), int64(start))
		if err != nil {
			return nil, err
		}

		for i := int(len(info.BlockMetas) - 1); i >= 0; i-- {
			if 0 == bytes.Compare(hash, info.BlockMetas[i].BlockID.Hash) {
				return DeprecatedGetBlockByNumber(blockStore, state, info.BlockMetas[i].Header.Height, full, readReceipts)
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
