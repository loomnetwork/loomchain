package handler

import (
	"bytes"
	"os"

	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
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
	testUpdateReceipts(t, ReceiptHandlerChain)

	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))
	testHandler(t, ReceiptHandlerLevelDb)

	os.RemoveAll(leveldb.Db_Filename)
	_, err = os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))
	testUpdateReceipts(t, ReceiptHandlerChain)
}

func testHandler(t *testing.T, v ReceiptHandlerVersion) {
	height := uint64(1)
	state := common.MockState(height)

	handler, err := NewReceiptHandler(v, &loomchain.DefaultEventHandler{}, DefaultMaxReceipts)
	require.NoError(t, err)

	var writer loomchain.WriteReceiptHandler
	writer = handler

	var receiptHandler loomchain.ReceiptHandler
	receiptHandler = handler

	var txHashList [][]byte
	for txNum := 0; txNum < 20; txNum++ {
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
		}
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
		require.EqualValues(t, index*2, receipt.TransactionIndex)
		if index == 5 {
			require.EqualValues(t, loomchain.StatusTxFail, receipt.Status)
		} else {
			require.EqualValues(t, loomchain.StatusTxSuccess, receipt.Status)
		}
	}

	err = receiptHandler.CommitBlock(state, int64(height))
	require.NoError(t, err)

	pendingHashList = reader.GetPendingTxHashList()
	require.EqualValues(t, 0, len(pendingHashList))

	for index, txHash := range txHashList {
		txReceipt, err := reader.GetReceipt(state, txHash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(txHash, txReceipt.TxHash))
		require.EqualValues(t, index*2, txReceipt.TransactionIndex)
		if index == 5 {
			require.EqualValues(t, loomchain.StatusTxFail, txReceipt.Status)
		} else {
			require.EqualValues(t, loomchain.StatusTxSuccess, txReceipt.Status)
		}
	}

	require.NoError(t, receiptHandler.Close())
	require.NoError(t, receiptHandler.ClearData())
}


// Same as testHandler, excepts saves mock transaction results in tendermint data objects.
// Then tests `updateReceipts` add corrected block hash and transaction ids to receipts.
func testUpdateReceipts(t *testing.T, v ReceiptHandlerVersion) {
	height := uint64(1)
	state := common.MockState(height)

	handler, err := NewReceiptHandler(v, &loomchain.DefaultEventHandler{}, DefaultMaxReceipts)
	require.NoError(t, err)

	var writer loomchain.WriteReceiptHandler
	writer = handler

	var receiptHandler loomchain.ReceiptHandler
	receiptHandler = handler

	var delieverTx []*abci.ResponseDeliverTx
	var txHashList [][]byte
	for txNum := 0; txNum < 20; txNum++ {
		var resp abci.ResponseDeliverTx

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
		delieverTx = append(delieverTx, &resp)
	}

	err = receiptHandler.CommitBlock(state, int64(height))
	require.NoError(t, err)

	blockHash := []byte("My block hash")
	handler.updateReceipts(state, delieverTx, blockHash)

	var reader loomchain.ReadReceiptHandler
	reader = handler
	for index, txHash := range txHashList {
		txReceipt, err := reader.GetReceipt(state, txHash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(blockHash, txReceipt.BlockHash))
		require.EqualValues(t, 0, bytes.Compare(txHash, txReceipt.TxHash))
		require.EqualValues(t, index, txReceipt.TransactionIndex)
	}

	require.NoError(t, receiptHandler.Close())
	require.NoError(t, receiptHandler.ClearData())
}