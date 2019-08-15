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
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
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
	blockStore store.BlockStore,
	state loomchain.ReadOnlyState,
	height int64,
	full bool,
	evmAuxStore *evmaux.EvmAuxStore,
) (resp eth.JsonBlockObject, err error) {
	// todo make information about pending block available
	if height > state.Block().Height {
		return resp, errors.New("get block information for pending blocks not implemented yet")
	}

	var blockResult *ctypes.ResultBlock
	blockResult, err = blockStore.GetBlockByHeight(&height)
	if err != nil {
		return resp, errors.Wrapf(err, "GetBlockByNumber failed to get block %d", height)
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
	bloomFilter := evmAuxStore.GetBloomFilter(uint64(height))
	blockInfo.LogsBloom = eth.EncBytes(bloomFilter)
	for index, tx := range blockResult.Block.Data.Txs {
		if full {
			txResult, err := blockStore.GetTxResult(tx.Hash())
			if err != nil {
				return resp, errors.Wrapf(err, "cant find tx details, hash %X", tx.Hash())
			}

			txObj, _, err := GetTxObjectFromBlockResult(blockResult, txResult, int64(index))
			if err != nil {
				return resp, errors.Wrapf(err, "cant resolve tx, hash %X", tx.Hash())
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

func GetTxObjectFromBlockResult(
	blockResult *ctypes.ResultBlock, txResult *ctypes.ResultTx, index int64,
) (eth.JsonTxObject, *eth.Data, error) {
	tx := blockResult.Block.Data.Txs[index]
	var contractAddress *eth.Data
	txObj := eth.JsonTxObject{
		BlockHash:        eth.EncBytes(blockResult.BlockMeta.BlockID.Hash),
		BlockNumber:      eth.EncInt(blockResult.Block.Header.Height),
		TransactionIndex: eth.EncInt(int64(index)),
		Value:            eth.EncInt(0),
		GasPrice:         eth.EncInt(0),
		Gas:              eth.EncInt(0),
		Hash:             eth.EncBytes(tx.Hash()),
	}

	var signedTx auth.SignedTx
	if err := proto.Unmarshal([]byte(tx), &signedTx); err != nil {
		return eth.GetEmptyTxObject(), nil, err
	}

	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return eth.GetEmptyTxObject(), nil, err
	}
	txObj.Nonce = eth.EncInt(int64(nonceTx.Sequence))

	var txTx loomchain.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &txTx); err != nil {
		return eth.GetEmptyTxObject(), nil, err
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(txTx.Data, &msg); err != nil {
		return eth.GetEmptyTxObject(), nil, err
	}
	txObj.From = eth.EncAddress(msg.From)

	var input []byte
	switch txTx.Id {
	case deployId:
		{
			var deployTx vm.DeployTx
			if err := proto.Unmarshal(msg.Data, &deployTx); err != nil {
				return eth.GetEmptyTxObject(), nil, err
			}
			input = deployTx.Code
			if deployTx.VmType == vm.VMType_EVM {
				var resp vm.DeployResponse
				if err := proto.Unmarshal(txResult.TxResult.Data, &resp); err != nil {
					return eth.GetEmptyTxObject(), nil, err
				}

				var respData vm.DeployResponseData
				if err := proto.Unmarshal(resp.Output, &respData); err != nil {
					return eth.GetEmptyTxObject(), nil, err
				}
				contractAddress = eth.EncPtrAddress(resp.Contract)
				if len(respData.TxHash) > 0 {
					txObj.Hash = eth.EncBytes(respData.TxHash)
				}
			}
			if deployTx.Value != nil {
				txObj.Value = eth.EncBigInt(*deployTx.Value.Value.Int)
			}
		}
	case callId:
		{
			var callTx vm.CallTx
			if err := proto.Unmarshal(msg.Data, &callTx); err != nil {
				return eth.GetEmptyTxObject(), nil, err
			}
			input = callTx.Input
			to := eth.EncAddress(msg.To)
			txObj.To = &to
			if callTx.VmType == vm.VMType_EVM && len(txResult.TxResult.Data) > 0 {
				txObj.Hash = eth.EncBytes(txResult.TxResult.Data)
			}
			if callTx.Value != nil {
				txObj.Value = eth.EncBigInt(*callTx.Value.Value.Int)
			}
		}
	case migrationTx:
		to := eth.EncAddress(msg.To)
		txObj.To = &to
		input = msg.Data
	default:
		return eth.GetEmptyTxObject(), nil, fmt.Errorf("unrecognised tx type %v", txTx.Id)
	}
	txObj.Input = eth.EncBytes(input)

	return txObj, contractAddress, nil
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
			if info.BlockMetas[i] != nil {
				if 0 == bytes.Compare(hash, info.BlockMetas[i].BlockID.Hash) {
					return info.BlockMetas[i].Header.Height, nil //    int64(int(end) + i), nil
				}
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
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *evmaux.EvmAuxStore,
) ([]byte, error) {
	var blockresult *ctypes.ResultBlock
	iHeight := height
	blockresult, err := blockStore.GetBlockByHeight(&iHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "DeprecatedGetBlockByNumber failed to get block %d", iHeight)
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
	blockinfo.LogsBloom = evmAuxStore.GetBloomFilter(uint64(height))

	txHashList, err := evmAuxStore.GetTxHashList(uint64(height))
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
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *evmaux.EvmAuxStore,
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
			if info.BlockMetas[i] != nil {
				if 0 == bytes.Compare(hash, info.BlockMetas[i].BlockID.Hash) {
					return DeprecatedGetBlockByNumber(blockStore, state, info.BlockMetas[i].Header.Height, full, readReceipts, evmAuxStore)
				}
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
