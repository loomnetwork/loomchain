package handler

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestReceiptsHandlerChain(t *testing.T) {
	testHandler(t, ReceiptHandlerChain)
	testHandler(t, ReceiptHandlerLevelDb)
}

func testHandler(t *testing.T, v ReceiptHandlerVersion) {
	height := uint64(1)
	state := common.MockState(height)
	
	handler := NewReceiptHandler(v,	&loomchain.DefaultEventHandler{})
	
	var writer loomchain.WriteReceiptHandler
	writer = handler
	var txHashList [][]byte
	for txNum := 0 ; txNum < 20 ; txNum++ {
		if txNum % 2 == 0 {
			stateI := common.MockStateTx(state, height, uint64(txNum))
			_, err := writer.CacheReceipt(stateI, addr1, addr2, []*loomchain.EventData{}, nil)
			require.NoError(t, err)
			_, err = writer.CacheReceipt(stateI, addr1, addr2, []*loomchain.EventData{}, nil)
			require.NoError(t, err)
			txHash, err := writer.CacheReceipt(stateI, addr1, addr2, []*loomchain.EventData{}, nil)
			require.NoError(t, err)
			txHashList = append(txHashList, txHash)
		}
	}
	
	require.EqualValues(t, int(19), len(handler.receiptsCache))
	
	var receiptHandler loomchain.ReceiptHandler
	receiptHandler = handler
	err := receiptHandler.CommitBlock(state, int64(height))
	require.NoError(t, err)
	
	var reader loomchain.ReadReceiptHandler
	reader = handler
	for index, txHash := range txHashList {
		txReceipt, err := reader.GetReceipt(state, txHash)
		require.NoError(t, err)
		require.EqualValues(t, string(txHash), string(txReceipt.TxHash))
		require.EqualValues(t, index*2, txReceipt.TransactionIndex)
	}
	
	receiptHandler.ClearData()
}