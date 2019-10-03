package evmaux

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	legacyEvmAuxDBName = "receipts_db"

	bloomPrefix     = []byte("bf")
	txHashPrefix    = []byte("th")
	txRefPrefix     = []byte("txr")
	dupTxHashPrefix = []byte("dtx")

	// keys to receipts linked list
	headKey          = []byte("leveldb:head")
	tailKey          = []byte("leveldb:tail")
	currentDbSizeKey = []byte("leveldb:size")

	ErrTxReceiptNotFound      = errors.New("Tx receipt not found")
	ErrPendingReceiptNotFound = errors.New("Pending receipt not found")
)

const (
	statusTxSuccess = int32(1)
)

func dupTxHashKey(txHash []byte) []byte {
	return util.PrefixKey(dupTxHashPrefix, txHash)
}

func bloomFilterKey(height uint64) []byte {
	return util.PrefixKey(bloomPrefix, blockHeightToBytes(height))
}

func evmTxHashKey(height uint64) []byte {
	return util.PrefixKey(txHashPrefix, blockHeightToBytes(height))
}

func blockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.BigEndian.PutUint64(heightB, height)
	return heightB
}

func renameReceiptsDB(path, newName string) error {
	if _, err := os.Stat(filepath.Join(path, legacyEvmAuxDBName)); !os.IsNotExist(err) {
		err := os.Rename(filepath.Join(path, legacyEvmAuxDBName), filepath.Join(path, newName+".db"))
		if err != nil {
			return err
		}
	}
	return nil
}

// ChildTxRef links a Tendermint tx hash to an EVM tx hash.
type ChildTxRef struct {
	ParentTxHash []byte
	ChildTxHash  []byte
}

type EvmAuxStore struct {
	db             dbm.DB
	maxReceipts    uint64
	dupEVMTxHashes map[string]bool // duplicate evm tx hashes list
}

func LoadStore(dbName, rootPath string, maxReceipts uint64) (*EvmAuxStore, error) {
	if maxReceipts == 0 {
		return NewEvmAuxStore(dbm.NewMemDB(), maxReceipts), nil
	}

	// if receipts_db exits, rename it to (default: evmaux.db)
	if err := renameReceiptsDB(rootPath, dbName); err != nil {
		return nil, err
	}
	evmAuxDB, err := dbm.NewGoLevelDB(dbName, rootPath)
	if err != nil {
		return nil, err
	}
	evmAuxStore := NewEvmAuxStore(evmAuxDB, maxReceipts)

	// load duplicate tx hashes from db
	dupEVMTxHashes := make(map[string]bool)
	iter := evmAuxDB.Iterator(
		dupTxHashPrefix, util.PrefixRangeEnd(dupTxHashPrefix),
	)
	defer iter.Close()
	for iter.Valid() {
		dupTxHash, err := util.UnprefixKey(iter.Key(), dupTxHashPrefix)
		if err != nil {
			return nil, err
		}
		dupEVMTxHashes[string(dupTxHash)] = true
		iter.Next()
	}
	evmAuxStore.SetDupEVMTxHashes(dupEVMTxHashes)

	return evmAuxStore, nil
}

func NewEvmAuxStore(db dbm.DB, maxReceipts uint64) *EvmAuxStore {
	return &EvmAuxStore{
		db:             db,
		maxReceipts:    maxReceipts,
		dupEVMTxHashes: make(map[string]bool),
	}
}

func (s *EvmAuxStore) SetDupEVMTxHashes(dupEVMTxHashes map[string]bool) {
	s.dupEVMTxHashes = dupEVMTxHashes
}

func (s *EvmAuxStore) GetDupEVMTxHashes() map[string]bool {
	return s.dupEVMTxHashes
}

func (s *EvmAuxStore) GetBloomFilter(height uint64) []byte {
	filter := s.db.Get(bloomFilterKey(height))
	if len(filter) == 0 {
		return nil
	}
	return filter
}

func (s *EvmAuxStore) GetTxHashList(height uint64) ([][]byte, error) {
	protHashList := s.db.Get(evmTxHashKey(height))
	if len(protHashList) == 0 {
		return [][]byte{}, nil
	}
	txHashList := types.EthTxHashList{}
	err := proto.Unmarshal(protHashList, &txHashList)
	return txHashList.EthTxHash, err
}

// SaveChildTxRefs persists references between Tendermint & EVM tx hashes to the underlying DB.
func (s *EvmAuxStore) SaveChildTxRefs(refs []ChildTxRef) error {
	if len(refs) == 0 {
		return nil
	}
	batch := s.db.NewBatch()
	defer batch.Close()
	for _, ref := range refs {
		batch.Set(util.PrefixKey(txRefPrefix, ref.ParentTxHash), ref.ChildTxHash)
	}
	batch.Write()
	return nil
}

// GetChildTxHash looks up the EVM tx hash that corresponds to the given Tendermint tx hash.
func (s *EvmAuxStore) GetChildTxHash(parentTxHash []byte) []byte {
	return s.db.Get(util.PrefixKey(txRefPrefix, parentTxHash))
}

// IsDupEVMTxHash checks if the tx hash is in the duplicate tx hash list
func (s *EvmAuxStore) IsDupEVMTxHash(txHash []byte) bool {
	_, ok := s.dupEVMTxHashes[string(txHash)]
	return ok
}

func (s *EvmAuxStore) DB() dbm.DB {
	return s.db
}

func (s *EvmAuxStore) GetReceipt(txHash []byte) (types.EvmTxReceipt, error) {
	txReceiptProto := s.db.Get(txHash)
	if len(txReceiptProto) == 0 {
		return types.EvmTxReceipt{}, ErrTxReceiptNotFound
	}
	txReceipt := types.EvmTxReceiptListItem{}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	if err != nil {
		return types.EvmTxReceipt{}, err
	}
	return *txReceipt.Receipt, nil
}

func (s *EvmAuxStore) CommitReceipts(receipts []*types.EvmTxReceipt, height uint64) error {
	if len(receipts) == 0 || s.maxReceipts == 0 {
		return nil
	}

	batch := s.db.NewBatch()

	size, headHash, tailHash, err := getDBParams(s.db)
	if err != nil {
		return errors.Wrap(err, "getting db params.")
	}

	tailReceiptItem := types.EvmTxReceiptListItem{}
	if len(headHash) > 0 {
		tailItemProto := s.db.Get(tailHash)
		if len(tailItemProto) == 0 {
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
			batch.Set(tailHash, protoTail)
			if !s.db.Has(tailHash) {
				size++
			}

		}

		// Set current receipt as next tail
		tailHash = txReceipt.TxHash
		tailReceiptItem = types.EvmTxReceiptListItem{Receipt: txReceipt, NextTxHash: nil}

		// only upload hashes to app db if transaction successful
		if txReceipt.Status == statusTxSuccess {
			txHashArray = append(txHashArray, txReceipt.TxHash)
		}

		events = append(events, txReceipt.Logs...)
	}

	if len(tailHash) > 0 {
		protoTail, err := proto.Marshal(&tailReceiptItem)
		if err != nil {
			log.Error(fmt.Sprintf("commit block receipts: marshal receipt item: %s", err.Error()))
		} else {
			batch.Set(tailHash, protoTail)
			if !s.db.Has(tailHash) {
				size++
			}
		}
	}
	setDBParams(batch, size, headHash, tailHash)

	// Set TxHashList
	// TODO: Get rid of this TxHashList because it is not correct anymore after child-tx-refs is used
	postTxHashList, err := proto.Marshal(&types.EthTxHashList{EthTxHash: txHashArray})
	if err != nil {
		return errors.Wrap(err, "marshal tx hash list")
	}
	batch.Set(evmTxHashKey(height), postTxHashList)
	// Set BloomFilter
	filter := bloom.GenBloomFilter(events)
	batch.Set(bloomFilterKey(height), filter)

	// Commit
	batch.WriteSync()

	return s.RemoveOldReceipts()
}

// RemoveOldReceipts removes old receipts once the number of receipts exceeds the limit
// TODO: This function is not used in the production since we keep all the receipts.
//       We should probaby get rid of it and the entire linked-list structure.
func (s *EvmAuxStore) RemoveOldReceipts() error {
	size, headHash, tailHash, err := getDBParams(s.db)
	if err != nil {
		return errors.Wrap(err, "getting db params.")
	}

	// do nothing if the number of receipts does not exceed the limit
	if s.maxReceipts >= size {
		return nil
	}

	batch := s.db.NewBatch()

	itemsDeleted := uint64(0)
	head := headHash
	toDeletedReceipt := size - s.maxReceipts
	for i := uint64(0); i < toDeletedReceipt && len(head) > 0; i++ {
		headItem := s.db.Get(head)
		txHeadReceiptItem := types.EvmTxReceiptListItem{}
		if err := proto.Unmarshal(headItem, &txHeadReceiptItem); err != nil {
			return errors.Wrapf(err, "unmarshal head %s", string(headItem))
		}
		batch.Delete(head)
		itemsDeleted++
		head = txHeadReceiptItem.NextTxHash
	}
	if itemsDeleted < toDeletedReceipt {
		return errors.Errorf("Unable to delete %v receipts, only %v deleted", toDeletedReceipt, itemsDeleted)
	}
	if size < itemsDeleted {
		return errors.Wrap(err, "invalid count of deleted receipts")
	}
	size -= itemsDeleted

	setDBParams(batch, size, head, tailHash)
	batch.WriteSync()
	return nil
}

func getDBParams(db dbm.DB) (size uint64, head, tail []byte, err error) {
	notEmpty := db.Has(currentDbSizeKey)
	if !notEmpty {
		return 0, []byte{}, []byte{}, nil
	}

	sizeB := db.Get(currentDbSizeKey)
	size = binary.LittleEndian.Uint64(sizeB)
	if size == 0 {
		return 0, []byte{}, []byte{}, nil
	}

	head = db.Get(headKey)
	if len(head) == 0 {
		return 0, []byte{}, []byte{}, errors.New("no head for non zero size receipt db")
	}

	tail = db.Get(tailKey)
	if len(tail) == 0 {
		return 0, []byte{}, []byte{}, errors.New("no tail for non zero size receipt db")
	}

	return size, head, tail, nil
}

func setDBParams(batch dbm.Batch, size uint64, head, tail []byte) {
	batch.Set(headKey, head)
	batch.Set(tailKey, tail)
	sizeB := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeB, size)
	batch.Set(currentDbSizeKey, sizeB)
}
