package handler

import (
	"bytes"
	"os"

	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	vtypes "github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestReceiptsHandlerChain(t *testing.T) {
	testHandler(t, ReceiptHandlerChain)

	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))
	testHandler(t, ReceiptHandlerLevelDb)
}

// Preform test block.
// 10 Evm transactions and 10 non EVM transactions
// First transaction an EVM deploy and tenth a failed EVM transactions.
// Mock move to next block, commit receitps and check consistency
func testHandler(t *testing.T, v ReceiptHandlerVersion) {
	height := uint64(1)
	state := common.MockState(height)

	handler, err := NewReceiptHandler(v, &loomchain.DefaultEventHandler{}, DefaultMaxReceipts)
	require.NoError(t, err)

	var writer loomchain.WriteReceiptHandler
	writer = handler

	var receiptHandler loomchain.ReceiptHandler
	receiptHandler = handler

	var delieverTxResponses []*abci.ResponseDeliverTx
	var txHashList [][]byte
	for txNum := 0; txNum < 20; txNum++ {
		var resp abci.ResponseDeliverTx
		loomchain.NewSequence(util.PrefixKey([]byte("nonce"), addr1.Bytes())).Next(state)
		if txNum%2 == 0 {
			stateI := common.MockStateTx(state, height, uint64(txNum))
			_, err = writer.CacheReceipt(stateI, addr1, addr2, []*loomchain.EventData{}, nil)
			require.NoError(t, err)
			txHash, err := writer.CacheReceipt(stateI, addr1, addr2, []*loomchain.EventData{}, nil)
			require.NoError(t, err)
			if txNum == 10 {
				receiptHandler.SetFailStatusCurrentReceipt()
			}
			receiptHandler.CommitCurrentReceipt()
			txHashList = append(txHashList, txHash)
			if txNum == 0 {
				createResp, err := proto.Marshal(&vm.DeployResponseData{
					TxHash:   txHash,
					Bytecode: []byte("some bytecode"),
				})
				require.NoError(t, err)
				response, err := proto.Marshal(&vtypes.DeployResponse{
					Output: createResp,
				})
				require.NoError(t, err)
				resp.Data = response
				resp.Info = utils.DeployEvm
			} else {
				resp.Data = txHash
				resp.Info = utils.CallEVM
			}
		} else {
			resp.Data = []byte("Go transaction results")
			resp.Info = utils.CallPlugin
		}
		delieverTxResponses = append(delieverTxResponses, &resp)
	}

	require.EqualValues(t, int(10), len(handler.receiptsCache))
	require.EqualValues(t, int(10), len(txHashList))

	var reader loomchain.ReadReceiptHandler
	reader = handler

	pendingHashList := reader.GetPendingTxHashList()
	require.EqualValues(t, 10, len(pendingHashList))

	for index, hash := range pendingHashList {
		receipt, err := reader.GetPendingReceipt(hash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(hash, receipt.TxHash))
		require.EqualValues(t, index*2+1, receipt.Nonce)
		if index == 5 {
			require.EqualValues(t, loomchain.StatusTxFail, receipt.Status)
		} else {
			require.EqualValues(t, loomchain.StatusTxSuccess, receipt.Status)
		}
	}

	blockHash := []byte("My block hash")
	height++
	state = common.MockStateAt(state, height)
	switch handler.v {
	case ReceiptHandlerChain:
		require.NoError(t, handler.chainReceipts.CommitBlock(state, handler.receiptsCache, uint64(height), blockHash))
	case ReceiptHandlerLevelDb:
		require.NoError(t, handler.leveldbReceipts.CommitBlock(state, handler.receiptsCache, uint64(height), blockHash))
	default:
		require.NoError(t, loomchain.ErrInvalidVersion)
	}
	handler.txHashList = [][]byte{}
	handler.receiptsCache = []*types.EvmTxReceipt{}

	txHashListLastBlock, err := common.GetTxHashList(state, uint64(height)-1)
	require.NoError(t, err)
	handler.confirmConsistancy(state, int64(height-1), delieverTxResponses, txHashListLastBlock, blockHash)

	pendingHashList = reader.GetPendingTxHashList()
	require.EqualValues(t, 0, len(pendingHashList))

	for index, txHash := range txHashList {
		txReceipt, err := reader.GetReceipt(state, txHash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(txHash, txReceipt.TxHash))
		if index < 5 {
			require.EqualValues(t, index, txReceipt.TransactionIndex)
			require.EqualValues(t, loomchain.StatusTxSuccess, txReceipt.Status)
		} else if index == 5 {
			require.EqualValues(t, 0, txReceipt.TransactionIndex)
			require.EqualValues(t, loomchain.StatusTxFail, txReceipt.Status)
		} else {
			require.EqualValues(t, index-1, txReceipt.TransactionIndex)
			require.EqualValues(t, loomchain.StatusTxSuccess, txReceipt.Status)
		}
	}

	require.NoError(t, receiptHandler.Close())
	require.NoError(t, receiptHandler.ClearData())
}
