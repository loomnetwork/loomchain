package leveldb

import (
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/receipts"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

const Db_Filename = "receipts_db"

func (wsr WriteLevelDbReceipts) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	txReceiptProto, err := wsr.db.Get(txHash, nil)
	if err != nil {
		return types.EvmTxReceipt{}, errors.Wrapf(err, "get receipt for %s", string(txHash))
	}
	txReceipt := types.EvmTxReceipt{}
	err = proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

/*
//TODO what to do here
func (wsr WriteLevelDbReceipts) Commit(txReceipt types.EvmTxReceipt) error {
	err := common.AppendTxHash(txReceipt.TxHash,wsr.State, uint64(txReceipt.BlockNumber))

*/

type WriteLevelDbReceipts struct {
	eventHandler loomchain.EventHandler

	lastNonce  uint64
	lastCaller loom.Address
	lastTxHash []byte
	db         *leveldb.DB
}

func NewWriteLevelDbReceipts(eventHandler loomchain.EventHandler) (*WriteLevelDbReceipts, error) {
	db, err := leveldb.OpenFile(Db_Filename, nil)
	if err != nil {
		return nil, errors.New("opening leveldb")
	}

	return &WriteLevelDbReceipts{
		db:           db,
		eventHandler: eventHandler,
	}, nil
}

func (wsr WriteLevelDbReceipts) Close() {
	if wsr.db != nil {
		wsr.db.Close()
	}
}

func (wsr WriteLevelDbReceipts) SaveEventsAndHashReceipt(state loomchain.State, caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	txReceipt, err := common.WriteReceipt(state, caller, addr, events, err, wsr.eventHandler)
	if err != nil {
		return []byte{}, errors.Wrap(err, "appending txHash to state")
	}

	height := common.BlockHeightToBytes(uint64(txReceipt.BlockNumber))
	bloomState := store.PrefixKVStore(receipts.BloomPrefix, state)
	bloomState.Set(height, txReceipt.LogsBloom)
	txHashState := store.PrefixKVStore(receipts.TxHashPrefix, state)
	txHashState.Set(height, txReceipt.TxHash)

	postTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		return nil, errors.Wrap(errMarshal, "marshal tx receipt")
	}

	nonce := auth.Nonce(state, caller)
	if nonce == wsr.lastNonce && 0 == caller.Compare(wsr.lastCaller) {
		wsr.db.Delete(wsr.lastTxHash, nil)
	}
	//TODO is this really needed?
	wsr.lastNonce = nonce
	wsr.lastCaller = caller
	wsr.lastTxHash = txReceipt.TxHash
	err = wsr.db.Put(txReceipt.TxHash, postTxReceipt, nil)

	return txReceipt.TxHash, err
}

func (wsr WriteLevelDbReceipts) ClearData() error {
	return os.RemoveAll(Db_Filename)
}

// Implement these functions
func (wsr WriteLevelDbReceipts) BeginTx() {
	panic("not implemented")
}

func (wsr WriteLevelDbReceipts) Rollback() { //this is a noop if the commit already happened
	panic("not implemented")
}

func (wsr WriteLevelDbReceipts) CommitFail() { //stores the failed tx, but assigns do an error status
	panic("not implemented")
}

func (wsr WriteLevelDbReceipts) Commit() {
	panic("not implemented")

	/*
	   +       receiptReader, err := r.ReceiptReaderFactory()(state)
	   +       if err != nil {
	   +               return errors.Wrap(err, "receipt reader")
	   +       }
	   +       txHashList, err := common.GetTxHashList(state, height)
	   +       if err != nil {
	   +               return errors.Wrap(err, "tx hash list")
	   +       }
	   +       var events []*types.EventData
	   +       for _, txHash := range txHashList {
	   +               txReceipt, err := receiptReader.GetReceipt(txHash)
	   +               if err != nil {
	   +                       return errors.Wrap(err, "get receipt")
	   +               }
	   +               events = append(events, txReceipt.Logs...)
	   +       }
	   +       filter := bloom.GenBloomFilter(events)
	   +       txHashState := store.PrefixKVStore(loomchain.BloomPrefix, state)
	   +       txHashState.Set(common.BlockHeightToBytes(height), filter)
	   +       return nil
	*/
}
