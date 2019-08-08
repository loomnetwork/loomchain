package leveldb

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	loom_types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/receipts/common"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
)

var (
	headKey          = []byte("leveldb:head")
	tailKey          = []byte("leveldb:tail")
	currentDbSizeKey = []byte("leveldb:size")
	tmHashPrefix     = []byte("leveldb:tmHashprefix")
)

func WriteReceipt(
	block loom_types.BlockHeader,
	caller, addr loom.Address,
	events []*types.EventData,
	status int32,
	eventHandler loomchain.EventHandler,
	evmTxIndex int32,
	nonce int64,
) (types.EvmTxReceipt, error) {
	txReceipt := types.EvmTxReceipt{
		Nonce:             nonce,
		TransactionIndex:  evmTxIndex,
		BlockHash:         block.CurrentHash,
		BlockNumber:       block.Height,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr.Local,
		LogsBloom:         bloom.GenBloomFilter(events),
		Status:            status,
		CallerAddress:     caller.MarshalPB(),
	}

	preTxReceipt, err := proto.Marshal(&txReceipt)
	if err != nil {
		return types.EvmTxReceipt{}, errors.Wrapf(err, "marshalling receipt")
	}
	h := sha256.New()
	h.Write(preTxReceipt)
	txHash := h.Sum(nil)

	txReceipt.TxHash = txHash
	txReceipt = AppendEvents(txReceipt, block, events, eventHandler)
	txReceipt.TransactionIndex = block.NumTxs - 1
	return txReceipt, nil
}

func AppendEvents(
	txReceipt types.EvmTxReceipt,
	block loom_types.BlockHeader,
	events []*types.EventData,
	eventHandler loomchain.EventHandler,
) types.EvmTxReceipt {
	for _, event := range events {
		event.TxHash = txReceipt.TxHash
		if eventHandler != nil {
			_ = eventHandler.Post(uint64(txReceipt.BlockNumber), event)
		}

		pEvent := types.EventData(*event)
		pEvent.BlockHash = block.CurrentHash
		pEvent.TransactionIndex = uint64(block.NumTxs - 1)
		txReceipt.Logs = append(txReceipt.Logs, &pEvent)
	}
	return txReceipt
}

func (lr *LevelDbReceipts) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	txReceiptProto, err := lr.evmAuxStore.DB().Get(txHash, nil)
	if err != nil {
		return types.EvmTxReceipt{}, errors.Wrapf(err, "get receipt for %s", string(txHash))
	}
	txReceipt := types.EvmTxReceiptListItem{}
	err = proto.Unmarshal(txReceiptProto, &txReceipt)
	return *txReceipt.Receipt, err
}

type LevelDbReceipts struct {
	MaxDbSize   uint64
	evmAuxStore *evmaux.EvmAuxStore
	tran        *leveldb.Transaction
}

func NewLevelDbReceipts(evmAuxStore *evmaux.EvmAuxStore, maxReceipts uint64) *LevelDbReceipts {
	return &LevelDbReceipts{
		MaxDbSize:   maxReceipts,
		evmAuxStore: evmAuxStore,
		tran:        nil,
	}
}

func (lr LevelDbReceipts) Close() error {
	if lr.evmAuxStore != nil {
		return lr.evmAuxStore.Close()
	}
	return nil
}

func (lr *LevelDbReceipts) GetHashFromTmHash(tmHash []byte) ([]byte, error) {
	db := lr.evmAuxStore.DB()
	return db.Get(util.PrefixKey(tmHashPrefix, tmHash), nil)
}

func (lr *LevelDbReceipts) saveTmHashIndex(tmHashIndex []common.HashPair) error {
	for _, pair := range tmHashIndex {
		key := util.PrefixKey(tmHashPrefix, pair.TmTxHash)
		if err := lr.tran.Put(key, pair.LoomTxHash, nil); err != nil {
			return err
		}
	}
	return nil
}

func (lr *LevelDbReceipts) CommitBlock(
	state loomchain.State,
	receipts []*types.EvmTxReceipt,
	height uint64,
	tmHashIndex []common.HashPair,
) error {
	if len(receipts) == 0 {
		return nil
	}

	size, headHash, tailHash, err := getDBParams(lr.evmAuxStore)
	if err != nil {
		return errors.Wrap(err, "getting db params.")
	}

	db := lr.evmAuxStore.DB()

	lr.tran, err = db.OpenTransaction()
	if err != nil {
		return errors.Wrap(err, "opening leveldb transaction")
	}
	defer lr.closeTransaction()

	if err := lr.saveTmHashIndex(tmHashIndex); err != nil {
		return err
	}

	tailReceiptItem := types.EvmTxReceiptListItem{}
	if len(headHash) > 0 {
		tailItemProto, err := lr.tran.Get(tailHash, nil)
		if err != nil {
			return errors.Wrap(err, "cannot find tail")
		}
		if err = proto.Unmarshal(tailItemProto, &tailReceiptItem); err != nil {
			return errors.Wrap(err, "unmarshalling tail")
		}
	}

	var txHashArray [][]byte
	events := make([]*types.EventData, 0, len(receipts))
	for _, txReceipt := range receipts {
		if txReceipt == nil || len(txReceipt.TxHash) == 0 {
			continue
		}

		// Update previous tail to point to current receipt
		if len(headHash) == 0 {
			headHash = txReceipt.TxHash
		} else {
			tailReceiptItem.NextTxHash = txReceipt.TxHash
			protoTail, err := proto.Marshal(&tailReceiptItem)
			if err != nil {
				log.Error(fmt.Sprintf("commit block receipts: marshal receipt item: %s", err.Error()))
				continue
			}
			updating, err := lr.tran.Has(tailHash, nil)
			if err != nil {
				return errors.Wrap(err, "cannot find tail hash")
			}

			if err := lr.tran.Put(tailHash, protoTail, nil); err != nil {
				log.Error(fmt.Sprintf("commit block receipts: put receipt in db: %s", err.Error()))
				continue
			} else if !updating {
				size++
			}
		}

		// Set current receipt as next tail
		tailHash = txReceipt.TxHash
		tailReceiptItem = types.EvmTxReceiptListItem{Receipt: txReceipt, NextTxHash: nil}

		// only upload hashes to app db if transaction successful
		if txReceipt.Status == common.StatusTxSuccess {
			txHashArray = append(txHashArray, txReceipt.TxHash)
		}

		events = append(events, txReceipt.Logs...)
	}
	if len(tailHash) > 0 {
		protoTail, err := proto.Marshal(&tailReceiptItem)
		if err != nil {
			log.Error(fmt.Sprintf("commit block receipts: marshal receipt item: %s", err.Error()))
		} else {
			updating, err := lr.tran.Has(tailHash, nil)
			if err != nil {
				return errors.Wrap(err, "cannot find tail hash")
			}
			if err := lr.tran.Put(tailHash, protoTail, nil); err != nil {
				log.Error(fmt.Sprintf("commit block receipts: putting receipt in db: %s", err.Error()))
			} else if !updating {
				size++
			}
		}
	}

	if lr.MaxDbSize < size {
		var numDeleted uint64
		headHash, numDeleted, err = removeOldEntries(lr.tran, headHash, size-lr.MaxDbSize)
		if err != nil {
			return errors.Wrap(err, "removing old receipts")
		}
		if size < numDeleted {
			return errors.Wrap(err, "invalid count of deleted receipts")
		}
		size -= numDeleted
	}
	if err := setDBParams(lr.tran, size, headHash, tailHash); err != nil {
		return errors.Wrap(err, "saving receipt db params")
	}

	filter := bloom.GenBloomFilter(events)
	if err := lr.evmAuxStore.SetTxHashList(lr.tran, txHashArray, height); err != nil {
		return errors.Wrap(err, "append tx list")
	}
	if err := lr.evmAuxStore.SetBloomFilter(lr.tran, filter, height); err != nil {
		return errors.Wrap(err, "set bloom filter")
	}

	if err := lr.tran.Commit(); err != nil {
		return errors.Wrap(err, "committing level db transaction")
	}
	lr.tran = nil
	return nil
}

func (lr *LevelDbReceipts) ClearData() {
	lr.evmAuxStore.ClearData()
}

func (lr *LevelDbReceipts) closeTransaction() {
	if lr.tran != nil {
		lr.tran.Discard()
		lr.tran = nil
	}
}

func removeOldEntries(tran *leveldb.Transaction, head []byte, number uint64) ([]byte, uint64, error) {
	itemsDeleted := uint64(0)
	for i := uint64(0); i < number && len(head) > 0; i++ {
		headItem, err := tran.Get(head, nil)
		if err != nil {
			return head, itemsDeleted, errors.Wrapf(err, "get head %s", string(head))
		}
		txHeadReceiptItem := types.EvmTxReceiptListItem{}
		if err := proto.Unmarshal(headItem, &txHeadReceiptItem); err != nil {
			return head, itemsDeleted, errors.Wrapf(err, "unmarshal head %s", string(headItem))
		}
		tran.Delete(head, nil)
		itemsDeleted++
		head = txHeadReceiptItem.NextTxHash
	}
	if itemsDeleted < number {
		return head, itemsDeleted, errors.Errorf("Unable to delete %v receipts, only %v deleted", number, itemsDeleted)
	}

	return head, itemsDeleted, nil
}

func getDBParams(db *evmaux.EvmAuxStore) (size uint64, head, tail []byte, err error) {
	notEmpty, err := db.DB().Has(currentDbSizeKey, nil)
	if err != nil {
		return size, head, tail, err
	}
	if !notEmpty {
		return 0, []byte{}, []byte{}, nil
	}

	sizeB, err := db.DB().Get(currentDbSizeKey, nil)
	if err != nil {
		return size, head, tail, err
	}
	size = binary.LittleEndian.Uint64(sizeB)
	if size == 0 {
		return 0, []byte{}, []byte{}, nil
	}

	head, err = db.DB().Get(headKey, nil)
	if err != nil {
		return size, head, tail, err
	}
	if len(head) == 0 {
		return 0, []byte{}, []byte{}, errors.New("no head for non zero size receipt db")
	}

	tail, err = db.DB().Get(tailKey, nil)
	if err != nil {
		return size, head, tail, err
	}
	if len(tail) == 0 {
		return 0, []byte{}, []byte{}, errors.New("no tail for non zero size receipt db")
	}

	return size, head, tail, nil
}

func setDBParams(tr *leveldb.Transaction, size uint64, head, tail []byte) error {
	if err := tr.Put(headKey, head, nil); err != nil {
		return err
	}

	if err := tr.Put(tailKey, tail, nil); err != nil {
		return err
	}

	sizeB := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeB, size)
	return tr.Put(currentDbSizeKey, sizeB, nil)
}
