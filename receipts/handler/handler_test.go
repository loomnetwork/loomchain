package handler

import (
	"bytes"
	"os"
	"testing"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
	"github.com/pkg/errors"
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

	// mock block
	for nonce := 0; nonce < 20; nonce++ {
		var txError error
		var resp abci.ResponseDeliverTx
		loomchain.NewSequence(util.PrefixKey([]byte("nonce"), addr1.Bytes())).Next(state)
		var txHash []byte

		if nonce%2 == 0 { // mock EVM transaction
			stateI := common.MockStateTx(state, height, uint64(nonce))
			_, err = writer.CacheReceipt(stateI, addr1, addr2, []*loomchain.EventData{}, nil)
			require.NoError(t, err)
			txHash, err = writer.CacheReceipt(stateI, addr1, addr2, []*loomchain.EventData{}, nil)
			require.NoError(t, err)
			if nonce == 18 { // mock error
				receiptHandler.SetFailStatusCurrentReceipt()
				txError = errors.New("Some EVM error")
			}
			if nonce == 0 { // mock deploy transaction
				resp.Data = []byte("proto with contract address and tx hash")
				resp.Info = utils.DeployEvm
			} else { // mock call transaction
				resp.Data = txHash
				resp.Info = utils.CallEVM
			}
		} else { // mock non-EVM transaction
			resp.Data = []byte("Go transaction results")
			resp.Info = utils.CallPlugin
		}

		// mock Application.processTx
		if txError != nil {
			receiptHandler.DiscardCurrentReceipt()
		} else {
			if resp.Info == utils.CallEVM || resp.Info == utils.DeployEvm {
				receiptHandler.CommitCurrentReceipt()
				txHashList = append(txHashList, txHash)
			}
		}

	}

	require.EqualValues(t, int(9), len(handler.receiptsCache))
	require.EqualValues(t, int(9), len(txHashList))

	var reader loomchain.ReadReceiptHandler
	reader = handler

	pendingHashList := reader.GetPendingTxHashList()
	require.EqualValues(t, 9, len(pendingHashList))

	for index, hash := range pendingHashList {
		receipt, err := reader.GetPendingReceipt(hash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(hash, receipt.TxHash))
		require.EqualValues(t, index*2+1, receipt.Nonce)
		require.EqualValues(t, index, receipt.TransactionIndex)
		require.EqualValues(t, common.StatusTxSuccess, receipt.Status)
	}

	err = receiptHandler.CommitBlock(state, int64(height))
	require.NoError(t, err)

	pendingHashList = reader.GetPendingTxHashList()
	require.EqualValues(t, 0, len(pendingHashList))

	for index, txHash := range txHashList {
		txReceipt, err := reader.GetReceipt(state, txHash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(txHash, txReceipt.TxHash))
		require.EqualValues(t, index*2+1, txReceipt.Nonce)
		require.EqualValues(t, index, txReceipt.TransactionIndex)
		require.EqualValues(t, common.StatusTxSuccess, txReceipt.Status)
	}

	require.NoError(t, receiptHandler.Close())
	require.NoError(t, receiptHandler.ClearData())
}
