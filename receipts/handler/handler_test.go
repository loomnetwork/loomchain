package handler

import (
	"bytes"
	"errors"
	"testing"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestReceiptsHandlerChain(t *testing.T) {
	height := uint64(1)
	state := common.MockState(height)

	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)

	handler := NewReceiptHandler(&loomchain.DefaultEventHandler{}, DefaultMaxReceipts, evmAuxStore)

	var writer loomchain.WriteReceiptHandler = handler
	var reader loomchain.ReadReceiptHandler = handler
	var receiptHandler loomchain.ReceiptHandlerStore = handler
	var txHashList [][]byte

	// mock block
	for nonce := 1; nonce < 21; nonce++ {
		var txError error
		var resp abci.ResponseDeliverTx
		loomchain.NewSequence(util.PrefixKey([]byte("nonce"), addr1.Bytes())).Next(state)
		var txHash []byte

		if nonce%2 == 1 { // mock EVM transaction
			stateI := common.MockStateTx(state, height, uint64(nonce))
			_, err = writer.CacheReceipt(stateI, addr1, addr2, []*types.EventData{}, nil, []byte{}, 1)
			require.NoError(t, err)
			txHash, err = writer.CacheReceipt(stateI, addr1, addr2, []*types.EventData{}, nil, []byte{}, 1)
			require.NoError(t, err)
			if nonce == 1 { // mock deploy transaction
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
		if nonce == 19 { // mock error
			currentReceipt := reader.GetCurrentReceipt()
			currentReceipt.Status = common.StatusTxFail
			txError = errors.New("Some EVM error")
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

	pendingHashList := reader.GetPendingTxHashList()
	require.EqualValues(t, 9, len(pendingHashList))

	for index, hash := range pendingHashList {
		receipt, err := reader.GetPendingReceipt(hash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(hash, receipt.TxHash))
		require.EqualValues(t, int64(index*2+1), receipt.Nonce)
		require.EqualValues(t, 2*index, receipt.TransactionIndex)
		require.EqualValues(t, common.StatusTxSuccess, receipt.Status)
	}

	err = receiptHandler.CommitBlock(int64(height))
	require.NoError(t, err)

	pendingHashList = reader.GetPendingTxHashList()
	require.EqualValues(t, 0, len(pendingHashList))

	for index, txHash := range txHashList {
		txReceipt, err := reader.GetReceipt(txHash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(txHash, txReceipt.TxHash))
		require.EqualValues(t, index*2+1, txReceipt.Nonce)
		require.EqualValues(t, 2*index, txReceipt.TransactionIndex)
		require.EqualValues(t, common.StatusTxSuccess, txReceipt.Status)
	}

	require.NoError(t, receiptHandler.Close())
	require.NoError(t, receiptHandler.ClearData())
}
