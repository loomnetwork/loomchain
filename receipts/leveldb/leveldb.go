package leveldb

import (
	"fmt"
	"os"
	
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/store"
	
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	Db_Filename = "receipts_db"
	Deault_DBHeight = 2000
)
var (
	levelDBPrefix   = []byte("receipt:leveldb")
	headKey         = []byte("leveldb:head")
	tailKey         = []byte("leveldb:tail")
)

func (lr* LevelDbReceipts) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	db, err := leveldb.OpenFile(Db_Filename, nil)
	defer db.Close()
	if err != nil {
		return types.EvmTxReceipt{}, errors.New("opening leveldb")
	}
	txReceiptProto, err := db.Get(txHash, nil)
	if err != nil {
		return types.EvmTxReceipt{}, errors.Wrapf(err,"get receipit for %s", string(txHash))
	}
	txReceipt := types.EvmTxReceiptListItem{}
	err = proto.Unmarshal(txReceiptProto, &txReceipt)
	return *txReceipt.Receipt, err
}

type LevelDbReceipts struct {
	MaxDbSize uint64
	currenctDbSize uint64
}

func (lr* LevelDbReceipts) CommitBlock(state loomchain.State, receipts []*types.EvmTxReceipt, height uint64) error  {
	var err error
	levelDBState := store.PrefixKVStore(levelDBPrefix, state)
	if  uint64(len(receipts)) >= lr.MaxDbSize {
		lr.ClearData()
	}
	db, err := leveldb.OpenFile(Db_Filename, nil)
	defer db.Close()
	if err != nil {
		return errors.Wrap(err, "opening leveldb")
	}
	headHash := levelDBState.Get(headKey)
	tailReceiptItem := types.EvmTxReceiptListItem{}
	tailHash := headHash
	if len(headHash) > 0 {
		tailHash = levelDBState.Get(tailKey)
		tailItemProto, err := db.Get(tailHash, nil)
		if err != nil  {
			log.Error(fmt.Sprintf("commit block receipts: cannot get tail: %s", err.Error()))
		} else if err = proto.Unmarshal(tailItemProto, &tailReceiptItem); err != nil {
			log.Error(fmt.Sprintf("commit block receipts: error unmarshalling tail: %s", err.Error()))
		}
	}

	defer levelDBState.Set(headKey, headHash)
	defer levelDBState.Set(tailKey, tailHash)
	
	var txHashArray [][]byte
	var events []*types.EventData

	for _, txReceipt := range receipts {
		if txReceipt == nil || len(txReceipt.TxHash) > 0 {
			continue
		}
		
		if len(headHash) == 0 {
			headHash = txReceipt.TxHash
		} else {
			tailReceiptItem.NextTxHash = txReceipt.TxHash
			protoTail, err := proto.Marshal(&tailReceiptItem)
			if err != nil {
				log.Error(fmt.Sprintf("commit block receipts: marshal receipt item: %s", err.Error()))
				continue
			}
			if err := db.Put(tailHash, protoTail, nil); err != nil {
				log.Error(fmt.Sprintf("commit block receipts: put receipt in db: %s", err.Error()))
			} else {
				lr.currenctDbSize = lr.currenctDbSize+1
			}
		}
		
		tailHash = txReceipt.TxHash
		txHashArray = append(txHashArray, (*txReceipt).TxHash)
		events = append(events, txReceipt.Logs...)
		tailReceiptItem = types.EvmTxReceiptListItem{txReceipt, nil}
	}
	if (len(tailHash) > 0){
		protoTail, err := proto.Marshal(&tailReceiptItem)
		if err != nil {
			log.Error(fmt.Sprintf("commit block receipts: marshal receipt item: %s", err.Error()))
		} else {
			db.Put(tailHash, protoTail, nil)
		}
	}
	
	if (lr.MaxDbSize < lr.currenctDbSize) {
		headHash, err = removeOldEntries(db, headHash, lr.currenctDbSize-lr.MaxDbSize)
		if err != nil {
			log.Error(fmt.Sprintf("commit block receipts: removing old rceipts: %s", err.Error()))
		}
	}
	
	if err := common.AppendTxHashList(state,txHashArray,  height); err != nil {
		return errors.Wrap(err, "append tx list")
	}
	filter := bloom.GenBloomFilter(events)
	common.SetBloomFilter(state, filter, height)
}

func (lr* LevelDbReceipts) ClearData() {
	os.RemoveAll(Db_Filename)
	lr.currenctDbSize = 0
}

func removeOldEntries(db *leveldb.DB, head []byte, number uint64) ([]byte, error) {
	for i := uint64(0) ; i < number && len(head) > 0 ; i++  {
		headItem, err := db.Get(head, nil)
		if err != nil {
			return head, errors.Wrapf(err,"get head %s", string(head))
		}
		txHeadReceiptItem := types.EvmTxReceiptListItem{}
		if err := proto.Unmarshal(headItem, &txHeadReceiptItem); err != nil {
			return head, errors.Wrapf(err,"unmarshl head %s",string(headItem))
		}
		db.Delete(head, nil)
		head = txHeadReceiptItem.NextTxHash
	}
	return head, nil
}


