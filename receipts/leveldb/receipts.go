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
		return types.EvmTxReceipt{}, errors.Wrapf(err, "get recipit for %s", string(txHash))
	}
	txReceipt := types.EvmTxReceipt{}
	err = proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

func (wsr WriteLevelDbReceipts) GetTxHash(state loomchain.ReadOnlyState, height uint64) ([]byte, error) {
	receiptState := store.PrefixKVReader(receipts.TxHashPrefix, state)
	txHash := receiptState.Get(common.BlockHeightToBytes(height))
	return txHash, nil
}

func (wsr WriteLevelDbReceipts) GetBloomFilter(state loomchain.ReadOnlyState, height uint64) ([]byte, error) {
	receiptState := store.PrefixKVReader(receipts.BloomPrefix, state)
	boomFilter := receiptState.Get(common.BlockHeightToBytes(height))
	return boomFilter, nil
}

type WriteLevelDbReceipts struct {
	EventHandler loomchain.EventHandler

	lastNonce  uint64
	lastCaller loom.Address
	lastTxHash []byte
	db         *leveldb.DB
}

func NewWriteLevelDbReceipts(EventHandler loomchain.EventHandler) (*WriteLevelDbReceipts, error) {
	db, err := leveldb.OpenFile(Db_Filename, nil)
	if err != nil {
		return nil, errors.New("opening leveldb")
	}

	return &WriteLevelDbReceipts{
		db: db,
	}, nil
}

func (wsr WriteLevelDbReceipts) Close() {
	if wsr.db != nil {
		wsr.db.Close()
	}
}

func (wsr WriteLevelDbReceipts) SaveEventsAndHashReceipt(state loomchain.State, caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	txReceipt, err := common.WriteReceipt(state, caller, addr, events, err, wsr.EventHandler)
	if err != nil {
		return []byte{}, err
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
