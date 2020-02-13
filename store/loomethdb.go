package store

import (
	"bytes"
	"log"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	LogEthDBBatch = true
	logger        log.Logger
	loggerStarted = false
)

// EthDBLogContext provides additional context when
type EthDBLogContext struct {
	blockHeight  int64
	contractAddr loom.Address
	callerAddr   loom.Address
}

func NewEthDBLogContext(height int64, contractAddr loom.Address, callerAddr loom.Address) *EthDBLogContext {
	return &EthDBLogContext{
		blockHeight:  height,
		contractAddr: contractAddr,
		callerAddr:   callerAddr,
	}
}

// LoomEthDB implements ethdb.Database
type LoomEthDB struct {
	db db.DBWrapper
}

func NewLoomEthDB(db db.DBWrapper) *LoomEthDB {
	return &LoomEthDB{
		db: db,
	}
}

func (s *LoomEthDB) Put(key []byte, value []byte) error {
	s.db.Set(util.PrefixKey(vmPrefix, key), value)
	return nil
}

func (s *LoomEthDB) Get(key []byte) ([]byte, error) {
	return s.db.Get(util.PrefixKey(vmPrefix, key)), nil
}

func (s *LoomEthDB) Has(key []byte) (bool, error) {
	return s.db.Has(util.PrefixKey(vmPrefix, key)), nil
}

func (s *LoomEthDB) Delete(key []byte) error {
	s.db.Delete(util.PrefixKey(vmPrefix, key))
	return nil
}

func (s *LoomEthDB) Close() error {
	return nil
}

func (s *LoomEthDB) NewBatch() ethdb.Batch {
	if LogEthDBBatch {
		return s.NewLogBatch(nil)
	}
	return newBatch(s.db)
}

// The methods below aren't used in loomchain so they're just stubs to satisfy the ethdb.Database interface

// Compact implements ethdb.Compacter
func (s *LoomEthDB) Compact(start []byte, limit []byte) error {
	panic("not implemented")
	return nil
}

// NewIterator implements ethdb.Iteratee
func (s *LoomEthDB) NewIterator() ethdb.Iterator {
	panic("not implemented")
	return nil
}

// NewIteratorWithStart implements ethdb.Iteratee
func (s *LoomEthDB) NewIteratorWithStart(start []byte) ethdb.Iterator {
	panic("not implemented")
	return nil
}

// NewIteratorWithPrefix implements ethdb.Iteratee
func (s *LoomEthDB) NewIteratorWithPrefix(prefix []byte) ethdb.Iterator {
	panic("not implemented")
	return nil
}

// Stat implements ethdb.Stater
func (s *LoomEthDB) Stat(property string) (string, error) {
	panic("not implemented")
	return "", nil
}

// HasAncient implements AncientReader
func (s *LoomEthDB) HasAncient(kind string, number uint64) (bool, error) {
	panic("not implemented")
	return false, nil
}

// Ancient implements AncientReader
func (s *LoomEthDB) Ancient(kind string, number uint64) ([]byte, error) {
	panic("not implemented")
	return nil, nil
}

// Ancients implements AncientReader
func (s *LoomEthDB) Ancients() (uint64, error) {
	panic("not implemented")
	return 0, nil
}

// AncientSize implements AncientReader
func (s *LoomEthDB) AncientSize(kind string) (uint64, error) {
	panic("not implemented")
	return 0, nil
}

// AppendAncient implements AncientWriter
func (s *LoomEthDB) AppendAncient(number uint64, hash, header, body, receipt, td []byte) error {
	panic("not implemented")
	return nil
}

// TruncateAncients implements AncientWriter
func (s *LoomEthDB) TruncateAncients(n uint64) error {
	panic("not implemented")
	return nil
}

// Sync implements AncientWriter
func (s *LoomEthDB) Sync() error {
	panic("not implemented")
	return nil
}

type batchOp struct {
	isDeleteOp bool
	key        []byte
	value      []byte
}

// implements ethdb.Batch
type batch struct {
	dbBatch dbm.Batch
	db      db.DBWrapper
	size    int
	ops     []batchOp
}

func newBatch(db db.DBWrapper) *batch {
	return &batch{
		dbBatch: db.NewBatch(),
		db:      db,
		ops:     nil,
	}
}

func (b *batch) Put(key, value []byte) error {
	b.dbBatch.Set(util.PrefixKey(vmPrefix, key), value)
	b.ops = append(b.ops, batchOp{false, key, value})
	b.size += len(value)
	return nil
}

func (b *batch) ValueSize() int {
	return b.size
}

func (b *batch) Write() error {
	b.dbBatch.Write()
	return nil
}

func (b *batch) Reset() {
	b.dbBatch.Close()
	b.dbBatch = b.db.NewBatch()
	b.size = 0
	b.ops = nil
}

func (b *batch) Delete(key []byte) error {
	b.dbBatch.Delete(util.PrefixKey(vmPrefix, key))
	b.ops = append(b.ops, batchOp{true, key, nil})
	return nil
}

func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	var err error
	for _, op := range b.ops {
		if err != nil {
			return err
		}
		if op.isDeleteOp {
			err = w.Delete(op.key)
		} else {
			err = w.Put(op.key, op.value)
		}
	}
	return err
}

type EthDBLogParams struct {
	LogFilename        string
	LogFlags           int
	LogReset           bool
	LogDelete          bool
	LogWrite           bool
	LogValueSize       bool
	LogPutKey          bool
	LogPutValue        bool
	LogPutDump         bool
	LogWriteDump       bool
	LogBeforeWriteDump bool
}

type kvPair struct {
	key   []byte
	value []byte
}
type LogBatch struct {
	db     db.DBWrapper
	size   int
	params EthDBLogParams
	cache  []kvPair
}

const batchHeaderWithContext = `

-----------------------------
| NEW BATCH
| Block: %v
| Contract: %v
| Caller: %v
-----------------------------

`

const batchHeader = `

-----------------------------
| NEW BATCH
-----------------------------

`

func (s *LoomEthDB) NewLogBatch(logContext *EthDBLogContext) ethdb.Batch {
	b := &LogBatch{
		db: s.db,
		params: EthDBLogParams{
			LogFilename:        "ethdb-batch.log",
			LogFlags:           0,
			LogReset:           true,
			LogDelete:          true,
			LogWrite:           true,
			LogValueSize:       false,
			LogPutKey:          true,
			LogPutValue:        false,
			LogPutDump:         false,
			LogWriteDump:       true,
			LogBeforeWriteDump: false,
		},
	}

	if !loggerStarted {
		file, err := os.Create(b.params.LogFilename)
		if err != nil {
			panic(err)
		}
		logger = *log.New(file, "", b.params.LogFlags)
		logger.Println("Created ethdb batch logger")
		loggerStarted = true
	}
	if logContext != nil {
		logger.Printf(batchHeaderWithContext, logContext.blockHeight, logContext.contractAddr, logContext.callerAddr)
	} else {
		logger.Print(batchHeader)
	}
	return b
}

func (b *LogBatch) Delete(key []byte) error {
	if b.params.LogDelete {
		logger.Println("Delete key: ", string(key))
	}
	b.cache = append(b.cache, kvPair{
		key:   common.CopyBytes(key),
		value: nil,
	})
	return nil
}

func (b *LogBatch) Put(key, value []byte) error {
	if b.params.LogPutKey {
		logger.Println("Put key: ", string(key))
	}
	if b.params.LogPutValue {
		logger.Println("Put value: ", string(value))
	}
	b.cache = append(b.cache, kvPair{
		key:   common.CopyBytes(key),
		value: common.CopyBytes(value),
	})
	b.size += len(value)
	if b.params.LogPutDump {
		b.Dump(&logger)
	}
	return nil
}

func (b *LogBatch) ValueSize() int {
	if b.params.LogValueSize {
		logger.Println("ValueSize : ", b.size)
	}
	return b.size
}

func (b *LogBatch) Write() error {
	if b.params.LogWrite {
		logger.Println("Write")
	}
	if b.params.LogBeforeWriteDump {
		logger.Println("Write, before : ")
		b.Dump(&logger)
	}

	sort.Slice(b.cache, func(j, k int) bool {
		return bytes.Compare(b.cache[j].key, b.cache[k].key) < 0
	})

	dbBatch := b.db.NewBatch()
	for _, kv := range b.cache {
		if kv.value == nil {
			dbBatch.Delete(util.PrefixKey(vmPrefix, kv.key))
		} else {
			dbBatch.Set(util.PrefixKey(vmPrefix, kv.key), kv.value)
		}
	}
	dbBatch.Write()

	if b.params.LogWriteDump {
		logger.Println("Write, after : ")
		b.Dump(&logger)
	}
	return nil
}

func (b *LogBatch) Reset() {
	if b.params.LogReset {
		logger.Println("Reset batch")
	}
	b.cache = make([]kvPair, 0)
	b.size = 0
}

func (b *LogBatch) Dump(logger *log.Logger) {
	logger.Print("\n---- BATCH DUMP ----\n")
	for i, kv := range b.cache {
		logger.Printf("IDX %d, KEY %s\n", i, kv.key)
	}
}

func (b *LogBatch) Replay(w ethdb.KeyValueWriter) error {
	var err error
	for _, kv := range b.cache {
		if err != nil {
			return err
		}
		if kv.value == nil {
			err = w.Delete(kv.key)
		} else {
			err = w.Put(kv.key, kv.value)
		}
	}
	return err
}
