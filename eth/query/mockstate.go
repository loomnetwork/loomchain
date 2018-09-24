// +build evm

package query

import (
	"context"
	"crypto/sha256"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	treceipts `github.com/loomnetwork/loomchain/receipts`
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/store"
	abci "github.com/tendermint/tendermint/abci/types"
)

type MockEvent struct {
	Address loom.LocalAddress
	Topics  []string
	Data    []byte
}

type MockReceipt struct {
	Height   uint64
	Contract loom.LocalAddress
	Events   []MockEvent
}

func MockPopulatedState(receipts []MockReceipt) (loomchain.State, error) {
	state := MockState()
	receiptState := store.PrefixKVStore(treceipts.ReceiptPrefix, state)
	txHashState := store.PrefixKVStore(treceipts.TxHashPrefix, state)
	bloomState := store.PrefixKVStore(treceipts.BloomPrefix, state)

	for _, mockR := range receipts {
		mockReciept := ptypes.EvmTxReceipt{
			TransactionIndex: 1,
			BlockNumber:      int64(mockR.Height),
			ContractAddress:  mockR.Contract,
		}
		protoTxReceipt, err := proto.Marshal(&mockReciept)
		if err != nil {
			return state, err
		}
		h := sha256.New()
		h.Write(protoTxReceipt)
		txHash := h.Sum(nil)
		mockReciept.TxHash = txHash

		events := []*loomchain.EventData{}
		for _, mockEvent := range mockR.Events {
			event := &loomchain.EventData{
				Topics:      mockEvent.Topics,
				BlockHeight: mockR.Height,
				EncodedBody: mockEvent.Data,
			}
			if len(mockEvent.Address) == 0 {
				event.Address = loom.Address{
					ChainID: "default",
					Local:   mockR.Contract,
				}.MarshalPB()
			} else {
				event.Address = loom.Address{
					ChainID: "default",
					Local:   mockEvent.Address,
				}.MarshalPB()
			}
			events = append(events, event)
			pEvent := ptypes.EventData(*event)
			mockReciept.Logs = append(mockReciept.Logs, &pEvent)
		}

		mockReciept.LogsBloom = GenBloomFilter(events)
		protoTxReceipt, err = proto.Marshal(&mockReciept)
		if err != nil {
			return state, err
		}

		height := utils.BlockHeightToBytes(mockR.Height)
		receiptState.Set(txHash, protoTxReceipt)
		bloomState.Set(height, mockReciept.LogsBloom)
		txHashState.Set(height, mockReciept.TxHash)
	}

	return state, nil
}

func MockState() loomchain.State {
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), abci.Header{})
}

func MockStateAt(state loomchain.State, newHeight int64) loomchain.State {
	header := abci.Header{}
	header.Height = newHeight
	return loomchain.NewStoreState(context.Background(), state, header)
}
