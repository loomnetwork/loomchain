package leveldb

import (
	"context"
	"os"
	"testing"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestReceipts(t *testing.T) {
	testEvents := []*loomchain.EventData{}
	eventHandler := loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher())

	caller1 := loom.Address{ChainID: "myChainID", Local: []byte("myCaller1")}
	addr1 := loom.Address{ChainID: "myChainID", Local: []byte("myContract1")}
	state1 := mockState(1)
	receiptWriter1, err := NewWriteLevelDbReceipts(eventHandler)
	require.NoError(t, err)
	txHash1, err := receiptWriter1.SaveEventsAndHashReceipt(state1, caller1, addr1, testEvents, nil)
	require.NoError(t, err)
	txHash, err := receiptWriter1.GetTxHash(state1, 1)
	require.NoError(t, err)
	require.Equal(t, string(txHash1), string(txHash))

	txReceipt1, err := receiptWriter1.GetReceipt(state1, txHash1)
	require.NoError(t, err)
	require.Equal(t, loom.UnmarshalAddressPB(txReceipt1.CallerAddress).String(), caller1.String())
	require.Equal(t, txReceipt1.BlockNumber, int64(1))
	require.Equal(t, string(txReceipt1.ContractAddress), string(addr1.Local))
	require.NoError(t, err)

	receiptWriter1.Close()

	caller2 := loom.Address{ChainID: "myChainID", Local: []byte("myCaller2")}
	addr2 := loom.Address{ChainID: "myChainID", Local: []byte("myContract2")}
	state2 := mockState(2)
	receiptWriter2, err := NewWriteLevelDbReceipts(eventHandler)
	require.NoError(t, err)
	txHash2, err := receiptWriter2.SaveEventsAndHashReceipt(state2, caller2, addr2, testEvents, nil)
	require.NoError(t, err)
	txHash, err = receiptWriter2.GetTxHash(state2, 2)
	require.NoError(t, err)
	require.Equal(t, string(txHash2), string(txHash))

	txReceipt2, err := receiptWriter2.GetReceipt(state2, txHash2)
	require.NoError(t, err)
	require.Equal(t, loom.UnmarshalAddressPB(txReceipt2.CallerAddress).String(), caller2.String())
	require.Equal(t, txReceipt2.BlockNumber, int64(2))
	require.Equal(t, string(txReceipt2.ContractAddress), string(addr2.Local))
	require.NoError(t, err)
	receiptWriter2.Close()

	_, err = os.Stat(Db_Filename)
	require.NoError(t, err)
	require.NoError(t, receiptWriter2.ClearData())
	_, err = os.Stat(Db_Filename)
	require.Error(t, err)
}

func mockState(height int64) loomchain.State {
	header := abci.Header{}
	header.Height = height
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), header)
}
