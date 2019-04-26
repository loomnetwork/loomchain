package store

import (
	"bytes"
	"log"
	"os"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/loomnetwork/go-loom"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	LogEvmStoreBatch = true
	logger           log.Logger
	loggerStarted    = false
)

type EvmStore interface {
	Delete([]byte) error
	Put([]byte, []byte) error
	Get([]byte) ([]byte, error)
	Has([]byte) (bool, error)
	Close()
	NewBatch() ethdb.Batch
}

var _ EvmStore = &KVEvmStore{}

type KVEvmStore struct {
	dbm.DB
	sync.Mutex
	lock       sync.RWMutex
	logContext *EvmStoreLogContext
}

type EvmStoreLogContext struct {
	BlockHeight  int64
	ContractAddr loom.Address
	CallerAddr   loom.Address
}

func NewKVEvmStore(db dbm.DB, logContext *EvmStoreLogContext) *KVEvmStore {
	return &KVEvmStore{
		DB:         db,
		logContext: logContext,
	}
}

func (EvmStore *KVEvmStore) Delete(key []byte) error {
	EvmStore.DB.Delete(key)
	return nil
}

func (EvmStore *KVEvmStore) Put(key []byte, value []byte) error {
	EvmStore.DB.Set(key, value)
	return nil
}

func (EvmStore *KVEvmStore) Get(key []byte) ([]byte, error) {
	return EvmStore.DB.Get(key), nil
}

func (EvmStore *KVEvmStore) Has(key []byte) (bool, error) {
	return EvmStore.DB.Has(key), nil
}

func (EvmStore *KVEvmStore) Close() {
}

func (EvmStore *KVEvmStore) NewBatch() ethdb.Batch {
	if LogEvmStoreBatch {
		return EvmStore.NewLogBatch(EvmStore.logContext)
	} else {
		newBatch := new(batch)
		newBatch.parentStore = EvmStore
		newBatch.Reset()
		return newBatch
	}
}

// implements ethdb.Batch
type kvPair struct {
	key   []byte
	value []byte
}

type batch struct {
	cache       []kvPair
	parentStore *KVEvmStore
	size        int
}

func (b *batch) Put(key, value []byte) error {
	b.cache = append(b.cache, kvPair{
		key:   common.CopyBytes(key),
		value: common.CopyBytes(value),
	})
	b.size += len(value)
	return nil
}

func (b *batch) ValueSize() int {
	return b.size
}

func (b *batch) Write() error {
	b.parentStore.lock.Lock()
	defer b.parentStore.lock.Unlock()

	sort.Slice(b.cache, func(j, k int) bool {
		return bytes.Compare(b.cache[j].key, b.cache[k].key) < 0
	})

	for _, kv := range b.cache {
		if kv.value == nil {
			b.parentStore.Delete(kv.key)
		} else {
			b.parentStore.Put(kv.key, kv.value)
		}
	}
	return nil
}

func (b *batch) Reset() {
	b.cache = make([]kvPair, 0)
	b.size = 0
}

func (b *batch) Delete(key []byte) error {
	b.cache = append(b.cache, kvPair{
		key:   common.CopyBytes(key),
		value: nil,
	})
	return nil
}

func (b *batch) Dump(logger *log.Logger) {
	b.parentStore.lock.Lock()
	defer b.parentStore.lock.Unlock()
	logger.Print("\n---- BATCH DUMP ----\n")
	for i, kv := range b.cache {
		logger.Printf("IDX %d, KEY %s\n", i, kv.key)
	}
}

type EVMLogParams struct {
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

type LogBatch struct {
	batch  batch
	params EVMLogParams
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

func (EvmStore *KVEvmStore) NewLogBatch(logContext *EvmStoreLogContext) ethdb.Batch {
	b := new(LogBatch)
	b.batch = *new(batch)
	b.batch.parentStore = EvmStore
	b.batch.Reset()
	b.params = EVMLogParams{
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
	}

	if !loggerStarted {
		file, err := os.Create(b.params.LogFilename)
		if err != nil {
			return &b.batch
		}
		logger = *log.New(file, "", b.params.LogFlags)
		logger.Println("Created ethdb batch logger")
		loggerStarted = true
	}
	if logContext != nil {
		logger.Printf(batchHeaderWithContext, logContext.BlockHeight, logContext.ContractAddr, logContext.CallerAddr)
	} else {
		logger.Print(batchHeader)
	}
	return b
}

func (b *LogBatch) Delete(key []byte) error {
	if b.params.LogDelete {
		logger.Println("Delete key: ", string(key))
	}
	return b.batch.Delete(key)
}

func (b *LogBatch) Put(key, value []byte) error {
	if b.params.LogPutKey {
		logger.Println("Put key: ", string(key))
	}
	if b.params.LogPutValue {
		logger.Println("Put value: ", string(value))
	}
	err := b.batch.Put(key, value)
	if b.params.LogPutDump {
		b.batch.Dump(&logger)
	}
	return err
}

func (b *LogBatch) ValueSize() int {
	size := b.batch.ValueSize()
	if b.params.LogValueSize {
		logger.Println("ValueSize : ", size)
	}
	return size
}

func (b *LogBatch) Write() error {
	if b.params.LogWrite {
		logger.Println("Write")
	}
	if b.params.LogBeforeWriteDump {
		logger.Println("Write, before : ")
		b.batch.Dump(&logger)
	}
	err := b.batch.Write()
	if b.params.LogWriteDump {
		logger.Println("Write, after : ")
		b.batch.Dump(&logger)
	}
	return err
}

func (b *LogBatch) Reset() {
	if b.params.LogReset {
		logger.Println("Reset batch")
	}
	b.batch.Reset()
}
