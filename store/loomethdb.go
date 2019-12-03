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
	store *EvmStore
}

func NewLoomEthDB(evmStore *EvmStore) *LoomEthDB {
	return &LoomEthDB{
		store: evmStore,
	}
}

func (s *LoomEthDB) Put(key []byte, value []byte) error {
	s.store.Set(util.PrefixKey(vmPrefix, key), value)
	return nil
}

func (s *LoomEthDB) Get(key []byte) ([]byte, error) {
	return s.store.Get(util.PrefixKey(vmPrefix, key)), nil
}

func (s *LoomEthDB) Has(key []byte) (bool, error) {
	return s.store.Has(util.PrefixKey(vmPrefix, key)), nil
}

func (s *LoomEthDB) Delete(key []byte) error {
	s.store.Delete(util.PrefixKey(vmPrefix, key))
	return nil
}

func (s *LoomEthDB) Close() {
}

func (s *LoomEthDB) NewBatch() ethdb.Batch {
	if LogEthDBBatch {
		return s.NewLogBatch(nil)
	}
	return newBatch(s.store)
}

// implements ethdb.Batch
type batch struct {
	dbBatch     dbm.Batch
	parentStore *EvmStore
	size        int
}

func newBatch(store *EvmStore) *batch {
	return &batch{
		dbBatch:     store.NewBatch(),
		parentStore: store,
	}
}

func (b *batch) Put(key, value []byte) error {
	b.dbBatch.Set(util.PrefixKey(vmPrefix, key), value)
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
	b.dbBatch = b.parentStore.NewBatch()
	b.size = 0
}

func (b *batch) Delete(key []byte) error {
	b.dbBatch.Delete(util.PrefixKey(vmPrefix, key))
	return nil
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
	parentStore *LoomEthDB
	size        int
	params      EthDBLogParams
	cache       []kvPair
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
	b := new(LogBatch)
	b.parentStore = s
	b.params = EthDBLogParams{
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
	size := b.size
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
		b.Dump(&logger)
	}

	sort.Slice(b.cache, func(j, k int) bool {
		return bytes.Compare(b.cache[j].key, b.cache[k].key) < 0
	})

	dbBatch := b.parentStore.store.NewBatch()
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
	b.cache = make([]kvPair, 0)
	b.size = 0
}

func (b *LogBatch) Dump(logger *log.Logger) {
	logger.Print("\n---- BATCH DUMP ----\n")
	for i, kv := range b.cache {
		logger.Printf("IDX %d, KEY %s\n", i, kv.key)
	}
}

// sortKeys sorts prefixed keys, it will sort the postfix of the key in ascending lexographical order
func sortKeys(prefix []byte, kvs []kvPair) []kvPair {
	var unsorted, sorted []int
	var tmpKv []kvPair
	for i, kv := range kvs {
		if 0 == bytes.Compare(prefix, kv.key[:len(prefix)]) {
			unsorted = append(unsorted, i)
			sorted = append(sorted, i)
		}
		tmpKv = append(tmpKv, kv)
	}
	sort.Slice(sorted, func(j, k int) bool {
		return bytes.Compare(kvs[sorted[j]].key, kvs[sorted[k]].key) < 0
	})
	for index := 0; index < len(sorted); index++ {
		kvs[unsorted[index]] = tmpKv[sorted[index]]
	}
	return kvs
}
