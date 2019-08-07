//nolint
package chain

import (
	"crypto/sha256"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/handler"
	"github.com/loomnetwork/loomchain/store"
)

// ReceiptHandler implements loomchain.ReadReceiptHandler, loomchain.WriteReceiptHandler, and
// loomchain.ReceiptHandlerStore interfaces in loom builds prior to 495.
type ReceiptHandler struct {
	eventHandler loomchain.EventHandler
}

func NewReceiptHandler(eventHandler loomchain.EventHandler) *ReceiptHandler {
	return &ReceiptHandler{
		eventHandler: eventHandler,
	}
}

func (r *ReceiptHandler) Version() handler.ReceiptHandlerVersion {
	return handler.ReceiptHandlerLegacyV2
}

func (r *ReceiptHandler) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	receiptState := store.PrefixKVReader(common.ReceiptPrefix, state)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := types.EvmTxReceipt{}
	if txReceiptProto == nil {
		return txReceipt, common.ErrTxReceiptNotFound
	}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err

}

func (r *ReceiptHandler) GetPendingReceipt(_ []byte) (types.EvmTxReceipt, error) {
	return types.EvmTxReceipt{}, errors.New("pending receipt not found")
}

func (r *ReceiptHandler) GetHashFromTmHash(_ []byte) ([]byte, error) {
	return nil, nil
}

func (r *ReceiptHandler) GetCurrentReceipt() *types.EvmTxReceipt {
	return nil
}

func (r *ReceiptHandler) GetPendingTxHashList() [][]byte {
	return nil
}

func (r *ReceiptHandler) GetTxHashList(_ uint64) ([][]byte, error) {
	return nil, nil
}

func (r *ReceiptHandler) Close() error {
	return nil
}

func (r *ReceiptHandler) ClearData() error {
	return nil
}

func (r *ReceiptHandler) CommitCurrentReceipt(_ []byte) {
}

func (r *ReceiptHandler) DiscardCurrentReceipt() {
}

func (r *ReceiptHandler) CommitBlock(_ loomchain.State, _ int64) error {
	return nil
}

func writeReceipt(
	state loomchain.State,
	caller, addr loom.Address,
	events []*types.EventData,
	err error,
	eventHadler loomchain.EventHandler,
) (types.EvmTxReceipt, error) {
	var status int32
	if err == nil {
		status = 1
	} else {
		status = 0
	}
	block := state.Block()
	txReceipt := types.EvmTxReceipt{
		TransactionIndex:  state.Block().NumTxs,
		BlockHash:         block.GetLastBlockID().Hash,
		BlockNumber:       state.Block().Height,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr.Local,
		LogsBloom:         bloom.GenBloomFilter(events),
		Status:            status,
		CallerAddress:     caller.MarshalPB(),
	}

	preTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return types.EvmTxReceipt{}, errors.Wrap(errMarshal, "marhsal tx receipt")
		} else {
			return types.EvmTxReceipt{}, errors.Wrapf(err, "marshalling receipt err %v", errMarshal)
		}
	}
	h := sha256.New()
	h.Write(preTxReceipt)
	txHash := h.Sum(nil)

	txReceipt.TxHash = txHash
	blockHeight := uint64(txReceipt.BlockNumber)
	for _, event := range events {
		event.TxHash = txHash
		if eventHadler != nil {
			_ = eventHadler.Post(blockHeight, event)
		}
		pEvent := types.EventData(*event)
		txReceipt.Logs = append(txReceipt.Logs, &pEvent)
	}

	return txReceipt, nil
}

func (r *ReceiptHandler) CacheReceipt(state loomchain.State, caller, addr loom.Address, events []*types.EventData, err error) ([]byte, error) {
	txReceipt, errWrite := writeReceipt(state, caller, addr, events, err, r.eventHandler)
	if errWrite != nil {
		if err == nil {
			return nil, errors.Wrap(errWrite, "writing receipt")
		} else {
			return nil, errors.Wrapf(err, "error writing receipt %v", errWrite)
		}
	}
	postTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return nil, errors.Wrap(errMarshal, "marhsal tx receipt")
		} else {
			return nil, errors.Wrapf(err, "marshalling receipt err %v", errMarshal)
		}
	}
	height := common.BlockHeightToBytes(uint64(txReceipt.BlockNumber))
	bloomState := store.PrefixKVStore(common.BloomPrefix, state)
	bloomState.Set(height, txReceipt.LogsBloom)
	txHashState := store.PrefixKVStore(common.TxHashPrefix, state)
	txHashState.Set(height, txReceipt.TxHash)

	receiptState := store.PrefixKVStore(common.ReceiptPrefix, state)
	receiptState.Set(txReceipt.TxHash, postTxReceipt)

	return txReceipt.TxHash, err
}

func (r *ReceiptHandler) SetFailStatusCurrentReceipt() {
}
