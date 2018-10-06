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

var (
	Db_Filename = "receipts_db"
	lastNonce   uint64
	lastCaller  loom.Address
	lastTxHash  []byte
)

type ReadLevelDbReceipts struct {
	State loomchain.ReadOnlyState
}

func (rsr ReadLevelDbReceipts) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	db, err := leveldb.OpenFile(Db_Filename, nil)
	defer db.Close()
	if err != nil {
		return types.EvmTxReceipt{}, errors.New("opening leveldb")
	}
	txReceiptProto, err := db.Get(txHash, nil)
	if err != nil {
		return types.EvmTxReceipt{}, errors.Wrapf(err, "get recipit for %s", string(txHash))
	}
	txReceipt := types.EvmTxReceipt{}
	err = proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

func (rsr ReadLevelDbReceipts) GetTxHash(height uint64) ([]byte, error) {
	receiptState := store.PrefixKVReader(receipts.TxHashPrefix, rsr.State)
	txHash := receiptState.Get(common.BlockHeightToBytes(height))
	return txHash, nil
}

func (rsr ReadLevelDbReceipts) GetBloomFilter(height uint64) ([]byte, error) {
	receiptState := store.PrefixKVReader(receipts.BloomPrefix, rsr.State)
	boomFilter := receiptState.Get(common.BlockHeightToBytes(height))
	return boomFilter, nil
}

type WriteLevelDbReceipts struct {
	State        loomchain.State
	EventHandler loomchain.EventHandler
}

func (wsr WriteLevelDbReceipts) SaveEventsAndHashReceipt(caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	txReceipt, err := common.WriteReceipt(wsr.State, caller, addr, events, err, wsr.EventHandler)
	if err != nil {
		return []byte{}, err
	}

	height := common.BlockHeightToBytes(uint64(txReceipt.BlockNumber))
	bloomState := store.PrefixKVStore(receipts.BloomPrefix, wsr.State)
	bloomState.Set(height, txReceipt.LogsBloom)
	txHashState := store.PrefixKVStore(receipts.TxHashPrefix, wsr.State)
	txHashState.Set(height, txReceipt.TxHash)

	postTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return nil, errors.Wrap(errMarshal, "marhsal tx receipt")
		} else {
			return nil, errors.Wrapf(err, "marshalling reciept err %v", errMarshal)
		}
	}
	db, err := leveldb.OpenFile(Db_Filename, nil)
	defer db.Close()
	if err != nil {
		return nil, errors.New("opening leveldb")
	}

	nonce := auth.Nonce(wsr.State, caller)
	if nonce == lastNonce && 0 == caller.Compare(lastCaller) {
		db.Delete(lastTxHash, nil)
	}
	lastNonce = nonce
	lastCaller = caller
	lastTxHash = txReceipt.TxHash
	err = db.Put(txReceipt.TxHash, postTxReceipt, nil)

	return txReceipt.TxHash, err
}

func (wsr WriteLevelDbReceipts) ClearData() error {
	return os.RemoveAll(Db_Filename)
}
