package receipts

import (
	`crypto/sha256`
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/loomchain`
	`github.com/loomnetwork/loomchain/eth/query`
	`github.com/loomnetwork/loomchain/eth/utils`
	`github.com/loomnetwork/loomchain/store`
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/gogo/protobuf/proto"
)

type StateReceipts struct {
	State loomchain.State
	EventHandler loomchain.EventHandler
}

func (sr StateReceipts) SaveEventsAndHashReceipt(caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	sState := *sr.State.(*loomchain.StoreState)
	ssBlock := sState.Block()
	var status int32
	if err == nil {
		status = 1
	} else {
		status = 0
	}
	txReceipt := types.EvmTxReceipt{
		TransactionIndex:  sState.Block().NumTxs,
		BlockHash:         ssBlock.GetLastBlockID().Hash,
		BlockNumber:       sState.Block().Height,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr.Local,
		LogsBloom:         query.GenBloomFilter(events),
		Status:            status,
		CallerAddress:     caller.MarshalPB(),
	}
	
	preTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return []byte{}, errMarshal
		} else {
			return []byte{}, err
		}
	}
	h := sha256.New()
	h.Write(preTxReceipt)
	txHash := h.Sum(nil)
	
	txReceipt.TxHash = txHash
	blockHeight := uint64(txReceipt.BlockNumber)
	for _, event := range events {
		event.TxHash = txHash
		_ = sr.EventHandler.Post(blockHeight, event)
		pEvent := types.EventData(*event)
		txReceipt.Logs = append(txReceipt.Logs, &pEvent)
	}
	
	postTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return []byte{}, errMarshal
		} else {
			return []byte{}, err
		}
	}
	
	receiptState := store.PrefixKVStore(utils.ReceiptPrefix, sr.State)
	receiptState.Set(txHash, postTxReceipt)
	
	height := utils.BlockHeightToBytes(blockHeight)
	bloomState := store.PrefixKVStore(utils.BloomPrefix, sr.State)
	bloomState.Set(height, txReceipt.LogsBloom)
	txHashState := store.PrefixKVStore(utils.TxHashPrefix, sr.State)
	txHashState.Set(height, txReceipt.TxHash)
	
	return txHash, err
}