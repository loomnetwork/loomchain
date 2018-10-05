package leveldb

import (
	`github.com/gogo/protobuf/proto`
	`github.com/loomnetwork/go-loom/plugin/types`
	`github.com/loomnetwork/loomchain`
	`github.com/loomnetwork/loomchain/receipts/common`
	`github.com/pkg/errors`
	`github.com/syndtr/goleveldb/leveldb`
	`os`
)

var (
	Db_Filename = "receipts_db"
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
		return types.EvmTxReceipt{}, errors.Wrapf(err,"get receipit for %s", string(txHash))
	}
	txReceipt := types.EvmTxReceipt{}
	err = proto.Unmarshal(txReceiptProto, &txReceipt)
	return txReceipt, err
}

type WriteLevelDbReceipts struct {
	State loomchain.State
}

func (wsr WriteLevelDbReceipts) Commit(txReceipt types.EvmTxReceipt) error {
	err := common.AppendTxHash(txReceipt.TxHash,wsr.State, uint64(txReceipt.BlockNumber))
	if err != nil {
		return errors.Wrap(err, "appending txHash to state")
	}

	db, err := leveldb.OpenFile(Db_Filename, nil)
	defer db.Close()
	if err != nil {
		return errors.New("opening leveldb")
	}
	
	postTxReceipt, err := proto.Marshal(&txReceipt)
	if err != nil {
		return errors.Wrap(err, "marshal tx receipt")
	}
	err = db.Put(txReceipt.TxHash, postTxReceipt, nil)
	if err != nil {
		return errors.Wrap(err, "add tx receipt to leveldb")
	}
	
	return nil
}

func (wsr WriteLevelDbReceipts) ClearData() error {
	return os.RemoveAll(Db_Filename)
}